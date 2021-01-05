package renamethis

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bkeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	log "github.com/sirupsen/logrus"
	amino "github.com/tendermint/go-amino"
	"golang.org/x/sync/semaphore"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/go-sdk/keys"
	kava "github.com/kava-labs/kava/app"
	bep3 "github.com/kava-labs/kava/x/bep3/types"
	tmlog "github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

// KavaClaimer is a worker that sends claim transactions on Kava
type KavaClaimer struct {
	Keybase keys.KeyManager
	Status  bool
}

func Main(ctx context.Context, c config.Config) {
	// Load kava claimers
	sdkConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(sdkConfig)
	cdc := kava.MakeCodec()

	// SETUP CLAIMERS --------------------------
	var kavaClaimers []KavaClaimer
	for _, kavaMnemonic := range c.Kava.Mnemonics {
		kavaClaimer := KavaClaimer{}
		keyManager, err := keys.NewMnemonicKeyManager(kavaMnemonic, kava.Bip44CoinType)
		if err != nil {
			log.Error(err)
		}
		kavaClaimer.Keybase = keyManager
		kavaClaimer.Status = true
		kavaClaimers = append(kavaClaimers, kavaClaimer)
	}

	// SETUP KAVA CLIENT --------------------------
	// Start Kava HTTP client
	http, err := rpcclient.New(c.Kava.Endpoint, "/websocket")
	if err != nil {
		panic(err)
	}
	http.Logger = tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	err = http.Start()
	if err != nil {
		panic(err)
	}

	// SETUP BNB CLIENT --------------------------
	// Set up Binance Chain client
	bncNetwork := btypes.TestNetwork
	if c.BinanceChain.ChainID == "Binance-Chain-Tigris" {
		bncNetwork = btypes.ProdNetwork
	}
	bnbClient := brpc.NewRPCClient(c.BinanceChain.Endpoint, bncNetwork)
	bnbKeyManager, err := bkeys.NewMnemonicKeyManager(c.BinanceChain.Mnemonic)
	if err != nil {
		panic(err)
	}
	bnbClient.SetKeyManager(bnbKeyManager)

	log.Info("Starting server...")
	claims := make(chan server.ClaimJob, 0)
	s := server.NewServer(claims)
	go s.StartServer()

	sem := semaphore.NewWeighted(int64(len(kavaClaimers)))

	// RUN WORKERS --------------------------
	for {
		select {
		case <-ctx.Done():
			return
		case claim := <-claims:
			switch strings.ToUpper(claim.TargetChain) {
			case server.TargetKava:
				if err := sem.Acquire(ctx, 1); err != nil {
					log.Error(err)
					return
				}

				go func() {
					defer sem.Release(1)
					Retry(10, 20*time.Second, func() (err ClaimError) {
						err = claimOnKava(c.Kava, http, claim, cdc, kavaClaimers)
						return
					})
				}()
				break
			case server.TargetBinance, server.TargetBinanceChain:
				go func() {
					Retry(10, 15*time.Second, func() (err ClaimError) {
						err = claimOnBinanceChain(bnbClient, claim)
						return
					})
				}()
				break
			}
		}
	}
}

func claimOnBinanceChain(bnbHTTP brpc.Client, claim server.ClaimJob) ClaimError {
	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		return NewErrorFailed(err)
	}

	swap, err := bnbHTTP.GetSwapByID(swapID[:])
	if err != nil {
		if strings.Contains(err.Error(), "zero records") {
			return NewErrorRetryable(fmt.Errorf("swap %s not found in state", claim.SwapID))
		}
		return NewErrorFailed(err)
	}

	status, err := bnbHTTP.Status()
	if err != nil {
		return NewErrorRetryable(err)
	}

	if swap.Status != btypes.Open || status.SyncInfo.LatestBlockHeight >= swap.ExpireHeight {
		return NewErrorFailed(fmt.Errorf("swap %s has status %s and cannot be claimed", claim.SwapID, swap.Status))
	}

	randomNumber, err := hex.DecodeString(claim.RandomNumber)
	if err != nil {
		return NewErrorFailed(err)
	}

	res, err := bnbHTTP.ClaimHTLT(swapID[:], randomNumber[:], brpc.Sync)
	if err != nil {
		return NewErrorFailed(err)
	}

	if res.Code != 0 {
		return NewErrorFailed(errors.New(res.Log))
	}

	log.Info("Claim tx sent to Binance Chain: ", res.Hash.String())
	return nil
}

func claimOnKava(config config.KavaConfig, http *rpcclient.HTTP, claim server.ClaimJob,
	cdc *amino.Codec, kavaClaimers []KavaClaimer) ClaimError {
	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		return NewErrorFailed(err)
	}

	err = isClaimableKava(http, cdc, swapID)
	if err != nil {
		return err
	}

	var claimer KavaClaimer
	selectedClaimer := false
	for !selectedClaimer {
		source := rand.NewSource(time.Now().UnixNano())
		r := rand.New(source)
		randNumb := r.Intn(len(kavaClaimers))
		randClaimer := kavaClaimers[randNumb%len(kavaClaimers)]
		if randClaimer.Status {
			selectedClaimer = true
			claimer = randClaimer
		}
	}

	fromAddr := claimer.Keybase.GetAddr()

	randomNumber, err := hex.DecodeString(claim.RandomNumber)
	if err != nil {
		return NewErrorFailed(err)
	}

	msg := bep3.NewMsgClaimAtomicSwap(fromAddr, swapID, randomNumber)
	if err := msg.ValidateBasic(); err != nil {
		return NewErrorFailed(fmt.Errorf("msg basic validation failed: \n%v", msg))
	}

	signMsg := &authtypes.StdSignMsg{
		ChainID:       config.ChainID,
		AccountNumber: 0,
		Sequence:      0,
		Fee:           authtypes.NewStdFee(250000, nil),
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}

	sequence, accountNumber, err := getKavaAcc(http, cdc, fromAddr)
	if err != nil {
		return NewErrorFailed(err)
	}
	signMsg.Sequence = sequence
	signMsg.AccountNumber = accountNumber

	signedMsg, err := claimer.Keybase.Sign(*signMsg, cdc)
	if err != nil {
		return NewErrorFailed(err)
	}
	tx := tmtypes.Tx(signedMsg)

	maxTxLength := 1024 * 1024
	if len(tx) > maxTxLength {
		return NewErrorFailed(fmt.Errorf("the tx data exceeds max length %d ", maxTxLength))
	}

	res, err := http.BroadcastTxSync(tx)
	if err != nil {
		return NewErrorFailed(err)
	}

	if res.Code != 0 {
		return NewErrorFailed(errors.New(res.Log))
	}

	log.Info("Claim tx sent to Kava: ", res.Hash.String())
	time.Sleep(7 * time.Second) // After sending the transaction, wait a full block
	return nil
}

// Check if swap is claimable
func isClaimableKava(http *rpcclient.HTTP, cdc *amino.Codec, swapID []byte) ClaimError {
	claimableParams := bep3.NewQueryAtomicSwapByID(swapID)
	claimableBz, err := cdc.MarshalJSON(claimableParams)
	if err != nil {
		return NewErrorFailed(err)
	}

	claimablePath := "custom/bep3/swap"

	result, err := http.ABCIQuery(claimablePath, claimableBz)
	if err != nil {
		return NewErrorRetryable(err)
	}

	resp := result.Response
	if !resp.IsOK() {
		if strings.Contains(resp.Log, "atomic swap not found") {
			return NewErrorRetryable(errors.New(resp.Log))
		}
		return NewErrorFailed(errors.New(resp.Log))
	}

	value := result.Response.GetValue()
	if len(value) == 0 {
		return NewErrorFailed(errors.New("no response value"))
	}

	var swap bep3.AtomicSwap
	err = cdc.UnmarshalJSON(value, &swap)
	if err != nil {
		return NewErrorFailed(err)
	}

	if swap.Status == bep3.NULL {
		return NewErrorRetryable(fmt.Errorf("swap %s not found in state", swapID))
	} else if swap.Status == bep3.Expired || swap.Status == bep3.Completed {
		return NewErrorFailed(fmt.Errorf("swap %s has status %s and cannot be claimed", hex.EncodeToString(swapID), swap.Status))
	}

	return nil
}

func getKavaAcc(http *rpcclient.HTTP, cdc *amino.Codec, fromAddr sdk.AccAddress) (uint64, uint64, ClaimError) {
	params := authtypes.NewQueryAccountParams(fromAddr)
	bz, err := cdc.MarshalJSON(params)
	if err != nil {
		return 0, 0, NewErrorFailed(err)
	}

	path := fmt.Sprintf("custom/acc/account/%s", fromAddr.String())

	result, err := http.ABCIQuery(path, bz)
	if err != nil {
		return 0, 0, NewErrorFailed(err)
	}

	resp := result.Response
	if !resp.IsOK() {
		return 0, 0, NewErrorFailed(errors.New(resp.Log))
	}

	value := result.Response.GetValue()
	if len(value) == 0 {
		return 0, 0, NewErrorFailed(errors.New("no response value"))
	}

	var acc authtypes.BaseAccount
	err = cdc.UnmarshalJSON(value, &acc)
	if err != nil {
		return 0, 0, NewErrorFailed(err)
	}

	if acc.Address.Empty() {
		return 0, 0, NewErrorFailed(errors.New("the signer account does not exist on kava"))
	}

	return acc.Sequence, acc.AccountNumber, nil
}
