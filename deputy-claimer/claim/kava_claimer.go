package claim

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
	bnbmsg "github.com/kava-labs/binance-chain-go-sdk/types/msg"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmtypes "github.com/tendermint/tendermint/types"
)

type KavaClaimer struct {
	kavaClient    KavaChainClient
	bnbClient     BnbChainClient
	mnemonics     []string
	bnbDeputyAddr types.AccAddress
}

func NewKavaClaimer(kavaRestURL, kavaRPCURL, bnbRPCURL string, bnbDeputyAddrString string, mnemonics []string) KavaClaimer {
	cdc := app.MakeCodec()
	bnbDeputyAddr, err := types.AccAddressFromBech32(bnbDeputyAddrString)
	if err != nil {
		panic(err)
	}
	return KavaClaimer{
		kavaClient:    NewMixedKavaClient(kavaRestURL, kavaRPCURL, cdc), // XXX hard dependency makes testing hard
		bnbClient:     NewRpcBNBClient(bnbRPCURL, bnbDeputyAddrString),
		mnemonics:     mnemonics,
		bnbDeputyAddr: bnbDeputyAddr,
	}
}
func (kc KavaClaimer) Run(ctx context.Context) { // XXX name should communicate this starts a goroutine
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// XXX G34 too many levels of abstraction
				log.Println("finding available deputy claims for kava")
				err := kc.fetchAndClaimSwaps()
				if err != nil {
					log.Printf("error fetching and claiming bnb swaps: %v\n", err)
				}
				time.Sleep(5 * time.Minute)
				continue
			}
		}
	}(ctx)
}

// XXX G30 functions should do one thing
// XXX G34 descend only one level of abstraction, several times over
// these also make it hard to test
func (kc KavaClaimer) fetchAndClaimSwaps() error {

	claimableSwaps, err := getClaimableKavaSwaps(kc.kavaClient, kc.bnbClient, kc.bnbDeputyAddr)
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

		go func(mnemonic string, mnemonics chan string, swap claimableSwap) {

			log.Printf("sending claim for kava swap id %s", swap.swapID)
			defer func() { mnemonics <- mnemonic }()

			txHash, err := constructAndSendClaim(kc.kavaClient, mnemonic, swap.swapID, swap.randomNumber)
			if err != nil {
				errs <- fmt.Errorf("could not submit claim: %w", err)
				return
			}
			err = Wait(15*time.Second, func() (bool, error) {
				res, err := kc.kavaClient.GetTxConfirmation(txHash)
				if err != nil {
					return false, nil
				}
				if res.TxResult.Code != 0 {
					return true, fmt.Errorf("kava tx rejected from chain: %s", res.TxResult.Log)
				}
				return true, nil
			})
			if err != nil {
				errs <- fmt.Errorf("could not get claim tx confirmation: %w", err)
				return
			}
		}(mnemonic, availableMnemonics, swap)
	}

	// wait for all go routines to finish
	for i := 0; i < len(kc.mnemonics); i++ {
		<-availableMnemonics
	}
	// report any errors // XXX C1 inappropriate information
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

type claimableSwap struct {
	swapID       tmbytes.HexBytes // XXX should define my own byte type to abstract the different ones each chain uses
	randomNumber tmbytes.HexBytes
}

func getClaimableKavaSwaps(kavaClient KavaChainClient, bnbClient BnbChainClient, bnbDeputyAddr types.AccAddress) ([]claimableSwap, error) {
	swaps, err := kavaClient.GetOpenOutgoingSwaps()
	if err != nil {
		return nil, fmt.Errorf("could not fetch open swaps: %w", err)
	}
	log.Printf("found %d open kava swaps", len(swaps))

	// filter out new swaps // XXX C1 inappropriate information // XXX G34 too many levels of abstraction
	var filteredSwaps bep3types.AtomicSwaps
	for _, s := range swaps {
		if time.Unix(s.Timestamp, 0).Add(10 * time.Minute).Before(time.Now()) { // XXX should abstract time to allow for easier testing
			filteredSwaps = append(filteredSwaps, s)
		}
	}

	// parse out swap ids, query those txs on bnb, extract random numbers
	var claimableSwaps []claimableSwap
	for _, s := range filteredSwaps {
		bID := bnbmsg.CalculateSwapID(s.RandomNumberHash, bnbDeputyAddr, s.Sender.String())
		randNum, err := bnbClient.GetRandomNumberFromSwap(bID)
		if err != nil {
			log.Printf("could not fetch random num for bnb swap ID %x: %v\n", bID, err)
			continue
		}
		claimableSwaps = append(
			claimableSwaps,
			claimableSwap{
				swapID:       s.GetSwapID(),
				randomNumber: randNum,
			})
	}
	return claimableSwaps, nil
}

func constructAndSendClaim(kavaClient KavaChainClient, mnemonic string, swapID, randNum tmbytes.HexBytes) ([]byte, error) {
	kavaKeyM, err := kavaKeys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
	if err != nil {
		return nil, fmt.Errorf("could not create key manager: %w", err)
	}
	// construct and sign tx
	msg := bep3types.NewMsgClaimAtomicSwap(kavaKeyM.GetAddr(), swapID, randNum)
	chainID, err := kavaClient.GetChainID()
	if err != nil {
		return nil, fmt.Errorf("could not fetch chain id: %w", err)
	}
	account, err := kavaClient.GetAccount(kavaKeyM.GetAddr())
	if err != nil {
		return nil, fmt.Errorf("could not fetch account: %w", err)
	}
	signMsg := authtypes.StdSignMsg{
		ChainID:       chainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
		Fee:           authtypes.NewStdFee(250_000, nil),
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}
	txBz, err := kavaKeyM.Sign(signMsg, kavaClient.GetCodec())
	if err != nil {
		return nil, fmt.Errorf("could not sign: %w", err)
	}
	// broadcast tx to mempool
	if err = kavaClient.BroadcastTx(txBz); err != nil {
		return nil, fmt.Errorf("could not submit claim: %w", err)
	}
	return tmtypes.Tx(txBz).Hash(), nil
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
