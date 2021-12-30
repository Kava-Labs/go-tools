package claim

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	bnbmsg "github.com/kava-labs/binance-chain-go-sdk/types/msg"
	tmtypes "github.com/kava-labs/tendermint/types"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/app/params"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
)

const (
	defaultGas       uint64        = 250_000
	kavaTxTimeout    time.Duration = 1 * time.Minute
	kavaLoopInterval time.Duration = 5 * time.Minute
)

var (
	defaultGasPrice sdk.DecCoin = sdk.NewDecCoinFromDec("ukava", sdk.MustNewDecFromStr("0.25"))
)

type KavaClaimError struct {
	Swap kavaClaimableSwap
	Err  error
}

func (r KavaClaimError) Error() string {
	return fmt.Sprintf("level=error msg=\"%s\" chain=%s srcSwapId=%s destSwapId=%s rndNum=%s amount=%s",
		r.Err,
		"kava",
		r.Swap.swapID,
		r.Swap.destSwapID,
		r.Swap.randomNumber,
		r.Swap.amount,
	)
}

type KavaClaimer struct {
	encodingConfig  params.EncodingConfig
	cdc             codec.Codec
	kavaClient      KavaChainClient
	bnbClient       BnbChainClient
	mnemonics       []string
	deputyAddresses DeputyAddresses
}

func NewKavaClaimer(
	kavaGrpcURL string,
	kavaGrpcEnableTLS bool,
	bnbRPCURL string,
	depAddrs DeputyAddresses,
	mnemonics []string,
) KavaClaimer {
	encodingConfig := app.MakeEncodingConfig()

	return KavaClaimer{
		encodingConfig:  encodingConfig,
		cdc:             encodingConfig.Marshaler,
		kavaClient:      NewGrpcKavaClient(kavaGrpcURL, kavaGrpcEnableTLS, encodingConfig.Marshaler),
		bnbClient:       NewRpcBNBClient(bnbRPCURL, depAddrs.AllBnb()),
		mnemonics:       mnemonics,
		deputyAddresses: depAddrs,
	}
}

func (kc KavaClaimer) Start(ctx context.Context) {
	go func(ctx context.Context) {
		nextPoll := time.After(0) // set wait to zero so it fires on startup
		for {
			select {
			case <-ctx.Done():
				return
			case <-nextPoll:
				log.Println("finding available deputy claims for kava")
				err := kc.fetchAndClaimSwaps()
				if err != nil {
					log.Printf("error fetching and claiming bnb swaps: %v\n", err)
				}
			}
			nextPoll = time.After(kavaLoopInterval)
		}
	}(ctx)
}

func (kc KavaClaimer) fetchAndClaimSwaps() error {
	claimableSwaps, err := getClaimableKavaSwaps(kc.kavaClient, kc.bnbClient, kc.deputyAddresses)
	if err != nil {
		return fmt.Errorf("could not fetch claimable swaps: %w", err)
	}
	log.Printf("found %d claimable kava HTLTs\n", len(claimableSwaps))

	// create and submit claim txs, distributing work over several addresses to avoid sequence number problems
	availableMnemonics := make(chan string, len(kc.mnemonics))
	for _, m := range kc.mnemonics {
		availableMnemonics <- m
	}
	errs := make(chan error, len(claimableSwaps))
	for _, swap := range claimableSwaps {
		mnemonic := <-availableMnemonics

		go func(mnemonic string, mnemonics chan string, swap kavaClaimableSwap) {

			log.Printf("sending claim for kava swap id %s", swap.swapID)
			defer func() { mnemonics <- mnemonic }()

			txHash, err := constructAndSendClaim(kc.kavaClient, kc.encodingConfig, mnemonic, swap.swapID, swap.randomNumber)
			if err != nil {
				errs <- KavaClaimError{Swap: swap, Err: fmt.Errorf("could not submit claim: %w", err)}
				return
			}
			err = Wait(kavaTxTimeout, func() (bool, error) {
				res, err := kc.kavaClient.GetTxConfirmation(txHash)
				if err != nil {
					return false, nil
				}
				if res.Code != 0 {
					return true, KavaClaimError{Swap: swap, Err: fmt.Errorf("kava tx rejected from chain: %s", res.Logs)}
				}
				return true, nil
			})
			if err != nil {
				errs <- KavaClaimError{Swap: swap, Err: fmt.Errorf("could not get claim tx confirmation: %w", err)}
				return
			}
		}(mnemonic, availableMnemonics, swap)
	}

	// wait for all go routines to finish
	for i := 0; i < len(kc.mnemonics); i++ {
		<-availableMnemonics
	}
	// return all errors
	var concatenatedErrs string
	close(errs)
	for e := range errs {
		concatenatedErrs += e.Error()
		concatenatedErrs += "\n"
	}
	if concatenatedErrs != "" {
		return fmt.Errorf("sending kava claims produced some errors: \n%s", concatenatedErrs)
	}
	return nil
}

func constructAndSendClaim(
	kavaClient KavaChainClient,
	encodingConfig params.EncodingConfig,
	mnemonic string,
	swapID, randNum tmbytes.HexBytes,
) ([]byte, error) {
	hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, 0)
	privKeyBytes, err := hd.Secp256k1.Derive()(mnemonic, "", hdPath.String())

	if err != nil {
		return nil, fmt.Errorf("could not derive private key bytes: %w", err)
	}

	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	accAddr := getAccAddress(privKey)

	// construct and sign tx
	msg := bep3types.NewMsgClaimAtomicSwap(accAddr.String(), swapID, randNum)
	chainID, err := kavaClient.GetChainID()
	if err != nil {
		return nil, fmt.Errorf("could not fetch chain id: %w", err)
	}

	account, err := kavaClient.GetAccount(accAddr)
	if err != nil {
		return nil, fmt.Errorf("could not fetch account: %w", err)
	}

	txBuilder := encodingConfig.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&msg)
	txBuilder.SetGasLimit(defaultGas)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(
		defaultGasPrice.Denom,
		defaultGasPrice.Amount.MulInt64(int64(defaultGas)).Ceil().TruncateInt(),
	)))

	signerData := authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: account.GetAccountNumber(),
		// Sequence:      account.GetSequence(),
	}

	signatureData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}
	sigV2 := signing.SignatureV2{
		PubKey:   privKey.PubKey(),
		Data:     &signatureData,
		Sequence: signerData.Sequence,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	signBytes, err := encodingConfig.TxConfig.SignModeHandler().GetSignBytes(signing.SignMode_SIGN_MODE_DIRECT, signerData, txBuilder.GetTx())
	if err != nil {
		return nil, err
	}
	signature, err := privKey.Sign(signBytes)
	if err != nil {
		return nil, err
	}

	sigV2.Data = &signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: signature,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, err
	}

	txBytes, err := encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, err
	}

	request := txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	}

	// broadcast tx to mempool
	if err = kavaClient.BroadcastTx(request); err != nil {
		return nil, fmt.Errorf("could not submit claim: %w", err)
	}

	tmtx := tmtypes.Tx(txBytes)
	tmTxHexBytes := tmbytes.HexBytes(tmtx.Hash())

	return tmTxHexBytes, nil
}

type kavaClaimableSwap struct {
	swapID       tmbytes.HexBytes
	destSwapID   tmbytes.HexBytes
	randomNumber tmbytes.HexBytes
	amount       sdk.Coins
}

func getClaimableKavaSwaps(
	kavaClient KavaChainClient,
	bnbClient BnbChainClient,
	depAddrs DeputyAddresses,
) ([]kavaClaimableSwap, error) {
	swaps, err := kavaClient.GetOpenOutgoingSwaps()
	if err != nil {
		return nil, fmt.Errorf("could not fetch open swaps: %w", err)
	}
	log.Printf("found %d open kava swaps", len(swaps))

	// filter out new swaps
	var filteredSwaps []bep3types.AtomicSwapResponse
	for _, s := range swaps {
		if time.Unix(s.Timestamp, 0).Add(10 * time.Minute).Before(time.Now()) {
			filteredSwaps = append(filteredSwaps, s)
		}
	}

	// parse out swap ids, query those txs on bnb, extract random numbers
	var claimableSwaps []kavaClaimableSwap
	for _, s := range filteredSwaps {
		bnbDeputyAddress, found := depAddrs.GetMatchingBnbStr(s.Recipient)
		if !found {
			log.Printf("unexpectedly could not find bnb deputy address for kava deputy %s", s.Recipient)
			continue
		}

		randomNumberHash, err := hex.DecodeString(s.RandomNumberHash)
		if err != nil {
			log.Printf("could not hex decode random number hash %v", s.RandomNumberHash)
			continue
		}

		bID := bnbmsg.CalculateSwapID(randomNumberHash, bnbDeputyAddress, s.Sender)
		randNum, err := bnbClient.GetRandomNumberFromSwap(bID)
		if err != nil {
			log.Printf("could not fetch random num for bnb swap ID %x: %v\n", bID, err)
			continue
		}

		swapID, err := hex.DecodeString(s.Id)
		if err != nil {
			log.Printf("could not hex decode swap ID %v", s.Id)
			continue
		}

		claimableSwaps = append(
			claimableSwaps,
			kavaClaimableSwap{
				swapID:       swapID,
				destSwapID:   bID,
				randomNumber: randNum,
				amount:       s.Amount,
			})
	}
	return claimableSwaps, nil
}

// Wait will poll the provided function until either:
//
// - it returns true
// - it returns an error
// - the timeout passes
func Wait(timeout time.Duration, shouldStop func() (bool, error)) error {
	endTime := time.Now().Add(timeout)

	for {
		stop, err := shouldStop()
		switch {
		case err != nil || stop:
			return err
		case time.Now().After(endTime):
			return errors.New("waiting timed out")
		}
		time.Sleep(1 * time.Millisecond)
	}
}

func getAccAddress(privKey cryptotypes.PrivKey) sdk.AccAddress {
	return privKey.PubKey().Address().Bytes()
}
