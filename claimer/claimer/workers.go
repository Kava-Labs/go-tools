package claimer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
	bep3 "github.com/kava-labs/kava/x/bep3/types"
	log "github.com/sirupsen/logrus"
	amino "github.com/tendermint/go-amino"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	tmtypes "github.com/tendermint/tendermint/types"
)

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

	res, err := bnbHTTP.ClaimHTLT(swapID[:], randomNumber[:], brpc.Commit)
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
	var randNum int
	selectedClaimer := false
	for !selectedClaimer {
		source := rand.NewSource(time.Now().UnixNano())
		r := rand.New(source)
		randNum = r.Intn(len(kavaClaimers))
		randClaimer := kavaClaimers[randNum]
		if randClaimer.Status {
			selectedClaimer = true
			kavaClaimers[randNum].Status = false
			claimer = randClaimer
		}
	}
	defer func() {
		kavaClaimers[randNum].Status = true
	}()

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
		Fee:           authtypes.NewStdFee(250000, sdk.Coins{}),
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

	res, err := http.BroadcastTxCommit(tx)
	if err != nil {
		return NewErrorFailed(err)
	}
	if res.CheckTx.Code != 0 {
		return NewErrorFailed(errors.New(res.CheckTx.Log))
	}
	if res.DeliverTx.Code != 0 {
		return NewErrorFailed(errors.New(res.DeliverTx.Log))
	}

	log.Info("Claim tx sent to Kava: ", res.Hash.String())
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
