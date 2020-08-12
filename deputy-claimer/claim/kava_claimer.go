package claim

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"
	bnbmsg "github.com/binance-chain/go-sdk/types/msg"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	authtypes "github.com/kava-labs/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	tmbytes "github.com/kava-labs/tendermint/libs/bytes"
	tmtypes "github.com/kava-labs/tendermint/types"
)

type KavaClaimer struct {
	kavaClient    kavaChainClient
	bnbClient     *bnbRpc.HTTP
	mnemonics     []string
	bnbDeputyAddr types.AccAddress
}

func NewKavaClaimer(kavaRestURL, kavaRPCURL, bnbRPCURL string, bnbDeputyAddrString string, mnemonics []string) KavaClaimer {
	cdc := kava.MakeCodec()
	bnbDeputyAddr, err := types.AccAddressFromBech32(bnbDeputyAddrString)
	if err != nil {
		panic(err)
	}
	return KavaClaimer{
		kavaClient:    newKavaChainClient(kavaRestURL, kavaRPCURL, cdc),
		bnbClient:     bnbRpc.NewRPCClient(bnbRPCURL, types.ProdNetwork),
		mnemonics:     mnemonics,
		bnbDeputyAddr: bnbDeputyAddr,
	}
}
func (kc KavaClaimer) Run(ctx context.Context) {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				log.Println("finding available deputy claims for kava")
				err := kc.fetchAndClaimSwaps()
				if err != nil {
					log.Println(err)
				}
				time.Sleep(5 * time.Minute)
				continue
			}
		}
	}(ctx)
}

func (kc KavaClaimer) fetchAndClaimSwaps() error {

	claimableSwaps, err := getClaimableKavaSwaps(kc.kavaClient, kc.bnbClient, kc.bnbDeputyAddr)
	if err != nil {
		return err
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
				errs <- err
				return
			}
			err = Wait(15*time.Second, func() (bool, error) {
				res, err := kc.kavaClient.getTxConfirmation(txHash)
				if err != nil {
					return false, nil
				}
				if res.TxResult.Code != 0 {
					return true, fmt.Errorf("tx rejected from chain: %s", res.TxResult.Log)
				}
				return true, nil
			})
			if err != nil {
				errs <- err
				return
			}
		}(mnemonic, availableMnemonics, swap)
	}

	// wait for all go routines to finish
	for i := 0; i > len(kc.mnemonics); i++ {
		<-availableMnemonics
	}
	// report any errors
	var concatenatedErrs string
	close(errs)
	for e := range errs {
		concatenatedErrs += e.Error()
		concatenatedErrs += "\n"
	}
	if concatenatedErrs != "" {
		return fmt.Errorf("sending claims produced some errors: \n%s", concatenatedErrs)
	}
	return nil
}

type claimableSwap struct {
	swapID       tmbytes.HexBytes
	randomNumber tmbytes.HexBytes
}

func getClaimableKavaSwaps(kavaClient kavaChainClient, bnbClient *bnbRpc.HTTP, bnbDeputyAddr types.AccAddress) ([]claimableSwap, error) {
	swaps, err := kavaClient.getOpenSwaps()
	if err != nil {
		return nil, err
	}

	// filter out new swaps
	var filteredSwaps bep3.AtomicSwaps
	for _, s := range swaps {
		if time.Unix(s.Timestamp, 0).Add(10 * time.Minute).Before(time.Now()) {
			filteredSwaps = append(filteredSwaps, s)
		}
	}

	// parse out swap ids, query those txs on bnb, extract random numbers
	var claimableSwaps []claimableSwap
	for _, s := range filteredSwaps {
		bID := bnbmsg.CalculateSwapID(s.RandomNumberHash, bnbDeputyAddr, s.Sender.String())
		bnbSwap, err := bnbClient.GetSwapByID(bID)
		if err != nil {
			return nil, err // TODO should not return on not found error
		}
		// check the bnb swap status is closed and has random number - ie it has been claimed
		if len(bnbSwap.RandomNumber) != 0 {
			claimableSwaps = append(
				claimableSwaps,
				claimableSwap{
					swapID:       s.GetSwapID(),
					randomNumber: tmbytes.HexBytes(bnbSwap.RandomNumber),
				})
		}
	}
	return claimableSwaps, nil
}

func constructAndSendClaim(kavaClient kavaChainClient, mnemonic string, swapID, randNum tmbytes.HexBytes) ([]byte, error) {
	kavaKeyM, err := kavaKeys.NewMnemonicKeyManager(mnemonic, kava.Bip44CoinType)
	if err != nil {
		return nil, err
	}
	// construct and sign tx
	msg := bep3.NewMsgClaimAtomicSwap(kavaKeyM.GetAddr(), swapID, randNum)
	chainID, err := kavaClient.getChainID()
	if err != nil {
		return nil, err
	}
	account, err := kavaClient.getAccount(kavaKeyM.GetAddr())
	if err != nil {
		return nil, err
	}
	signMsg := authtypes.StdSignMsg{
		ChainID:       chainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
		Fee:           authtypes.NewStdFee(250000, nil),
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}
	txBz, err := kavaKeyM.Sign(signMsg, kavaClient.codec)
	if err != nil {
		return nil, err
	}
	// broadcast tx to mempool
	if err = kavaClient.broadcastTx(txBz); err != nil {
		return nil, err
	}
	return tmtypes.Tx(txBz).Hash(), nil
}

// Wait will poll the provided function until either:
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
