package claim

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/binance-chain-go-sdk/keys"
	bnbmsg "github.com/kava-labs/binance-chain-go-sdk/types/msg"
	bnbtx "github.com/kava-labs/binance-chain-go-sdk/types/tx"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	tmtypes "github.com/kava-labs/tendermint/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
)

const (
	bnbTxTimeout    time.Duration = 1 * time.Minute
	bnbLoopInterval time.Duration = 5 * time.Minute
)

type BnbClaimError struct {
	Swap bnbClaimableSwap
	Err  error
}

func (r BnbClaimError) Error() string {
	return fmt.Sprintf("level=error msg=\"%s\" chain=%s srcSwapId=%s destSwapId=%s rndNum=%s amount=%s",
		r.Err,
		"binance",
		r.Swap.swapID,
		r.Swap.destSwapID,
		r.Swap.randomNumber,
		r.Swap.amount,
	)
}

type BnbClaimer struct {
	kavaClient      KavaChainClient
	bnbClient       BnbChainClient
	mnemonics       []string
	deputyAddresses DeputyAddresses
}

func NewBnbClaimer(
	kavaGrpcURL string,
	bnbRPCURL string,
	depAddrs DeputyAddresses,
	mnemonics []string,
) BnbClaimer {
	encodingConfig := app.MakeEncodingConfig()

	return BnbClaimer{
		kavaClient:      NewGrpcKavaClient(kavaGrpcURL, encodingConfig.Marshaler),
		bnbClient:       NewRpcBNBClient(bnbRPCURL, depAddrs.AllBnb()),
		mnemonics:       mnemonics,
		deputyAddresses: depAddrs,
	}
}
func (bc BnbClaimer) Start(ctx context.Context) {
	go func(ctx context.Context) {
		nextPoll := time.After(0) // set wait to zero so it fires on startup
		for {
			select {
			case <-ctx.Done():
				return
			case <-nextPoll:
				log.Println("finding available deputy claims for bnb")
				err := bc.fetchAndClaimSwaps()
				if err != nil {
					log.Printf("error fetching and claiming bnb swaps: %v\n", err)
				}
			}
			nextPoll = time.After(bnbLoopInterval)
		}
	}(ctx)
}

func (bc BnbClaimer) fetchAndClaimSwaps() error {
	claimableSwaps, err := getClaimableBnbSwaps(bc.kavaClient, bc.bnbClient, bc.deputyAddresses)
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

		go func(mnemonic string, mnemonics chan string, swap bnbClaimableSwap) {

			log.Printf("sending claim for bnb swap id %s", swap.swapID)
			defer func() { mnemonics <- mnemonic }()

			txHash, err := constructAndSendBnbClaim(bc.bnbClient, mnemonic, swap.swapID, swap.randomNumber)
			if err != nil {
				errs <- BnbClaimError{Swap: swap, Err: fmt.Errorf("could not submit claim: %w", err)}
				return
			}
			err = Wait(bnbTxTimeout, func() (bool, error) {
				res, err := bc.bnbClient.GetTxConfirmation(txHash)
				if err != nil {
					return false, nil
				}
				if res.TxResult.Code != 0 {
					return true, BnbClaimError{Swap: swap, Err: fmt.Errorf("bnb tx rejected from chain: %s", res.TxResult.Log)}
				}
				return true, nil
			})
			if err != nil {
				errs <- BnbClaimError{Swap: swap, Err: fmt.Errorf("could not get claim tx confirmation: %w", err)}
				return
			}
		}(mnemonic, availableMnemonics, swap)
	}

	// wait for all go routines to finish
	for i := 0; i < len(bc.mnemonics); i++ {
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

type bnbClaimableSwap struct {
	swapID       tmbytes.HexBytes
	destSwapID   tmbytes.HexBytes
	randomNumber tmbytes.HexBytes
	amount       types.Coins
}

func getClaimableBnbSwaps(kavaClient KavaChainClient, bnbClient BnbChainClient, depAddrs DeputyAddresses) ([]bnbClaimableSwap, error) {
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
	var claimableSwaps []bnbClaimableSwap
	for _, s := range filteredSwaps {
		kavaDeputyAddress, found := depAddrs.GetMatchingKava(s.To)
		if !found {
			log.Printf("unexpectedly could not find bnb deputy address for kava deputy %s", s.To)
			continue
		}
		kID := bep3types.CalculateSwapID(s.RandomNumberHash, kavaDeputyAddress, s.From.String())
		// get the random number for a claim transaction for the kava swap
		randNum, err := kavaClient.GetRandomNumberFromSwap(kID)
		if err != nil {
			log.Printf("could not fetch random num for kava swap ID %x: %v\n", kID, err)
			continue
		}
		claimableSwaps = append(
			claimableSwaps,
			bnbClaimableSwap{
				swapID:       bnbmsg.CalculateSwapID(s.RandomNumberHash, s.From, kavaDeputyAddress.String()),
				destSwapID:   kID,
				randomNumber: randNum,
				amount:       s.OutAmount,
			})
	}
	return claimableSwaps, nil
}

func constructAndSendBnbClaim(bnbClient BnbChainClient, mnemonic string, swapID, randNum tmbytes.HexBytes) ([]byte, error) {
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("could not create key manager: %w", err)
	}
	msg := bnbmsg.NewClaimHTLTMsg(
		keyManager.GetAddr(),
		swapID,
		randNum,
	)
	account, err := bnbClient.GetAccount(keyManager.GetAddr())
	if err != nil {
		return nil, fmt.Errorf("could not fetch account: %w", err)
	}
	signMsg := bnbtx.StdSignMsg{
		ChainID:       bnbClient.GetChainID(),
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
		Memo:          "",
		Msgs:          []bnbmsg.Msg{msg},
		Source:        bnbtx.Source,
	}
	txBz, err := keyManager.Sign(signMsg)
	if err != nil {
		return nil, fmt.Errorf("could not sign: %w", err)
	}
	err = bnbClient.BroadcastTx(txBz)
	if err != nil {
		return nil, fmt.Errorf("could not submit claim: %w", err)
	}
	return tmtypes.Tx(txBz).Hash(), nil
}
