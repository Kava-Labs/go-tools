package claimer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/go-sdk/keys"
	bep3 "github.com/kava-labs/kava/x/bep3/types"
	log "github.com/sirupsen/logrus"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

const (
	// ClaimTxDefaultGas is the gas limit to use for claim txs.
	// On kava-4, claim txs have historically reached up to 163072 gas.
	ClaimTxDefaultGas = 200_000

	// tendermintRPCCommitTimeoutErrorMsg is part of the error msg returned from a BroadcastTxCommit request.
	// It is triggered when the tx takes too long to make it into a block.
	// The timeout is defined in tendermint config, under rpc.timeout_broadcast_tx_commit
	tendermintRPCCommitTimeoutErrorMsg = "timed out waiting for tx to be included in a block"
)

func claimOnBinanceChain(bnbHTTP brpc.Client, claim server.ClaimJob) error {
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

func claimOnKava(config config.KavaConfig, client *KavaClient, claim server.ClaimJob, keyManager keys.KeyManager) error {
	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		return NewErrorFailed(err)
	}

	err = isClaimableKava(client, swapID)
	if err != nil {
		return err
	}

	fromAddr := keyManager.GetAddr()

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
		Fee:           authtypes.NewStdFee(ClaimTxDefaultGas, nil),
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}

	sequence, accountNumber, err := getAccountNumbers(client, fromAddr)
	if err != nil {
		return NewErrorFailed(err)
	}
	signMsg.Sequence = sequence
	signMsg.AccountNumber = accountNumber

	signedMsg, err := keyManager.Sign(*signMsg, client.cdc)
	if err != nil {
		return NewErrorFailed(err)
	}
	tx := tmtypes.Tx(signedMsg)

	maxTxLength := 1024 * 1024
	if len(tx) > maxTxLength {
		return NewErrorFailed(fmt.Errorf("the tx data exceeds max length %d ", maxTxLength))
	}

	res, err := client.BroadcastTxCommit(tx)
	if err != nil {
		if broadcastErrorIsRetryable(err) {
			return NewErrorRetryable(err)
		}
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

func broadcastErrorIsRetryable(err error) bool {
	var httpClientError *url.Error
	isRetryable :=
		// retry if the server times out waiting for a block
		strings.Contains(err.Error(), tendermintRPCCommitTimeoutErrorMsg) ||
			// retry if there's an error in posting the tx
			errors.As(err, &httpClientError)
	return isRetryable
}

// Check if swap is claimable
func isClaimableKava(client *KavaClient, swapID []byte) error {
	swap, err := client.GetAtomicSwap(swapID)
	if err != nil {
		return err // TODO
	}
	if swap.Status == bep3.NULL {
		return NewErrorRetryable(fmt.Errorf("swap %s not found in state", swapID))
	} else if swap.Status == bep3.Expired || swap.Status == bep3.Completed {
		return NewErrorFailed(fmt.Errorf("swap %s has status %s and cannot be claimed", hex.EncodeToString(swapID), swap.Status))
	}
	return nil
}

func getAccountNumbers(client *KavaClient, fromAddr sdk.AccAddress) (uint64, uint64, error) {

	acc, err := client.GetAccount(fromAddr)
	if err != nil {
		return 0, 0, err // TODO error parsing
	}

	return acc.GetSequence(), acc.GetAccountNumber(), nil
}
