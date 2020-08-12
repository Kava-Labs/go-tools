package claim

import (
	"context"
	"fmt"
	"log"
	"time"

	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"
	"github.com/binance-chain/go-sdk/keys"
	"github.com/binance-chain/go-sdk/types/msg"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	tmbytes "github.com/kava-labs/tendermint/libs/bytes"
)

type BnbClaimer struct {
	kavaClient     kavaChainClient
	bnbClient      bnbChainClient
	mnemonics      []string
	kavaDeputyAddr sdk.AccAddress
}

func NewBnbClaimer(kavaRestURL, kavaRPCURL, bnbRPCURL string, kavaDeputyAddrString, bnbDeputyAddrString string, mnemonics []string) BnbClaimer {
	cdc := kava.MakeCodec()
	kavaDeputyAddr, err := sdk.AccAddressFromBech32(kavaDeputyAddrString)
	if err != nil {
		panic(err)
	}
	return BnbClaimer{
		kavaClient:     newKavaChainClient(kavaRestURL, kavaRPCURL, cdc),
		bnbClient:      newBnbChainClient(bnbRPCURL, bnbDeputyAddrString),
		mnemonics:      mnemonics,
		kavaDeputyAddr: kavaDeputyAddr,
	}
}
func (bc BnbClaimer) Run(ctx context.Context) {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				log.Println("finding available deputy claims for bnb")
				err := bc.fetchAndClaimSwaps()
				if err != nil {
					log.Println(err)
				}
				time.Sleep(5 * time.Minute)
				continue
			}
		}
	}(ctx)
}

func (bc BnbClaimer) fetchAndClaimSwaps() error {

	claimableSwaps, err := getClaimableBnbSwaps(bc.kavaClient, bc.bnbClient, bc.kavaDeputyAddr)
	if err != nil {
		return err
	}
	log.Printf("found %d claimable bnb HTLTs\n", len(claimableSwaps))

	// create and submit claim txs, distributing work over several addresses to avoid sequence number problems
	availableMnemonics := make(chan string, len(bc.mnemonics))
	for _, m := range bc.mnemonics {
		availableMnemonics <- m
	}
	errs := make(chan error, len(claimableSwaps))
	for _, swap := range claimableSwaps {
		mnemonic := <-availableMnemonics

		go func(mnemonic string, mnemonics chan string, swap claimableSwap) {

			log.Printf("sending claim for bnb swap id %s", swap.swapID)
			defer func() { mnemonics <- mnemonic }()

			txHash, err := constructAndSendBnbClaim(bc.bnbClient, mnemonic, swap.swapID, swap.randomNumber)
			if err != nil {
				errs <- err
				return
			}
			err = Wait(15*time.Second, func() (bool, error) {
				res, err := bc.bnbClient.getTxConfirmation(txHash)
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
	for i := 0; i > len(bc.mnemonics); i++ {
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

func getClaimableBnbSwaps(kavaClient kavaChainClient, bnbClient bnbChainClient, kavaDeputyAddr sdk.AccAddress) ([]claimableSwap, error) {
	swaps, err := bnbClient.getOpenSwaps()
	if err != nil {
		return nil, err
	}

	// filter out new swaps
	var filteredSwaps []types.AtomicSwap
	for _, s := range swaps {
		if time.Unix(s.Timestamp, 0).Add(10 * time.Minute).Before(time.Now()) {
			filteredSwaps = append(filteredSwaps, s)
		}
	}
	// parse out swap ids, query those txs on bnb, extract random numbers
	var claimableSwaps []claimableSwap
	for _, s := range filteredSwaps {
		kID := bep3.CalculateSwapID(s.RandomNumberHash, kavaDeputyAddr, s.From.String())
		// get the random number for a claim transaction for the kava swap
		randNum, err := kavaClient.getRandomNumberFromSwap(kID)
		if err == nil {
			claimableSwaps = append(
				claimableSwaps,
				claimableSwap{
					swapID:       msg.CalculateSwapID(s.RandomNumberHash, s.From, kavaDeputyAddr.String()),
					randomNumber: randNum,
				})
		}
	}
	return claimableSwaps, nil
}

func constructAndSendBnbClaim(bnbClient bnbChainClient, mnemonic string, swapID, randNum tmbytes.HexBytes) ([]byte, error) {
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		return nil, err
	}
	bnbClient.bnbSDKClient.SetKeyManager(keyManager)
	defer bnbClient.bnbSDKClient.SetKeyManager(nil)
	res, err := bnbClient.bnbSDKClient.ClaimHTLT(swapID, randNum, bnbRpc.Sync)
	if err != nil {
		return nil, err
	}
	return res.Hash, nil
}
