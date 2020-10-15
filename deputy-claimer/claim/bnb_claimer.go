package claim

import (
	"context"
	"fmt"
	"log"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/binance-chain-go-sdk/types/msg"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
)

type BnbClaimer struct {
	kavaClient     KavaChainClient
	bnbClient      BnbChainClient
	mnemonics      []string
	kavaDeputyAddr sdk.AccAddress
}

func NewBnbClaimer(kavaRestURL, kavaRPCURL, bnbRPCURL string, kavaDeputyAddrString, bnbDeputyAddrString string, mnemonics []string) BnbClaimer {
	cdc := app.MakeCodec()
	kavaDeputyAddr, err := sdk.AccAddressFromBech32(kavaDeputyAddrString)
	if err != nil {
		panic(err)
	}
	return BnbClaimer{
		kavaClient:     NewMixedKavaClient(kavaRestURL, kavaRPCURL, cdc),
		bnbClient:      NewRpcBNBClient(bnbRPCURL, bnbDeputyAddrString),
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
					log.Printf("error fetching and claiming bnb swaps: %v\n", err)
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
		return fmt.Errorf("could not fetch claimable swaps: %w", err)
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
				errs <- fmt.Errorf("could not submit claim: %w", err)
				return
			}
			err = Wait(15*time.Second, func() (bool, error) {
				res, err := bc.bnbClient.GetTxConfirmation(txHash)
				if err != nil {
					return false, nil
				}
				if res.TxResult.Code != 0 {
					return true, fmt.Errorf("bnb tx rejected from chain: %s", res.TxResult.Log)
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
		return fmt.Errorf("sending bnb claims produced some errors: \n%s", concatenatedErrs)
	}
	return nil
}

func getClaimableBnbSwaps(kavaClient KavaChainClient, bnbClient BnbChainClient, kavaDeputyAddr sdk.AccAddress) ([]claimableSwap, error) {
	swaps, err := bnbClient.GetOpenOutgoingSwaps()
	if err != nil {
		return nil, fmt.Errorf("could not fetch open swaps: %w", err)
	}
	log.Printf("found %d open bnb swaps", len(swaps))

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
		kID := bep3types.CalculateSwapID(s.RandomNumberHash, kavaDeputyAddr, s.From.String())
		// get the random number for a claim transaction for the kava swap
		randNum, err := kavaClient.GetRandomNumberFromSwap(kID)
		if err != nil {
			log.Printf("could not fetch random num for kava swap ID %x: %v\n", kID, err)
			continue
		}
		claimableSwaps = append(
			claimableSwaps,
			claimableSwap{
				swapID:       msg.CalculateSwapID(s.RandomNumberHash, s.From, kavaDeputyAddr.String()),
				randomNumber: randNum,
			})
	}
	return claimableSwaps, nil
}

func constructAndSendBnbClaim(bnbClient BnbChainClient, mnemonic string, swapID, randNum tmbytes.HexBytes) ([]byte, error) {
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("could not create key manager: %w", err)
	}
	bnbClient.GetBNBSDKClient().SetKeyManager(keyManager) // XXX G14 feature envy
	defer bnbClient.GetBNBSDKClient().SetKeyManager(nil)
	res, err := bnbClient.GetBNBSDKClient().ClaimHTLT(swapID, randNum, bnbRpc.Sync)
	if err != nil {
		return nil, fmt.Errorf("could not submit claim: %w", err)
	}
	return res.Hash, nil
}
