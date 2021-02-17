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
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

const (
	// ClaimTxDefaultGas is the gas limit to use for claim txs.
	// On kava-4, claim txs have historically reached up to 163072 gas.
	ClaimTxDefaultGas = 200_000

	// TxConfirmationTimeout is the longest time to wait for a tx confirmation before giving up
	TxConfirmationTimeout      = 3 * 60 * time.Second
	TxConfirmationPollInterval = 2 * time.Second
)

var (
	// DefaultGasPrice is default fee to pay for a tx, per gas
	DefaultGasPrice sdk.DecCoin = sdk.NewDecCoinFromDec("ukava", sdk.MustNewDecFromStr("0.25"))
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

	swap, err := client.GetAtomicSwap(swapID)
	if err != nil {
		return NewErrorRetryable(err)
	}
	if swap.Status == bep3.NULL {
		return NewErrorRetryable(fmt.Errorf("swap %s not found in state", swapID))
	} else if swap.Status == bep3.Expired || swap.Status == bep3.Completed {
		return NewErrorFailed(fmt.Errorf("swap %s has status %s and cannot be claimed", hex.EncodeToString(swapID), swap.Status))
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
		Fee:           calculateFee(ClaimTxDefaultGas, DefaultGasPrice),
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

	res, err := client.BroadcastTxSync(tx)
	if err != nil {
		if broadcastErrorIsRetryable(err) {
			return NewErrorRetryable(err)
		}
		return NewErrorFailed(err)
	}
	if res.Code != 0 {
		if errorCodeIsRetryable(res.Codespace, res.Code) {
			return NewErrorRetryable(fmt.Errorf("tx rejected from mempool: %s", res.Log))
		}
		return NewErrorFailed(fmt.Errorf("tx rejected from mempool: %s", res.Log))
	}
	err = pollWithBackoff(TxConfirmationTimeout, TxConfirmationPollInterval, func() (bool, error) {
		log.WithFields(logrus.Fields{"swapID": claim.SwapID}).Debug("checking for tx confirmation") // TODO use non global logger, with swap ID field already included
		queryRes, err := client.GetTxConfirmation(res.Hash)
		if err != nil {
			return false, nil // poll again, it can't find the tx or node is down/slow
		}
		if queryRes.TxResult.Code != 0 {
			return true, fmt.Errorf("tx rejected from block: %s", queryRes.TxResult.Log) // return error, found tx but it didn't work
		}
		return true, nil // return nothing, found successfully confirmed tx
	})
	if err != nil {
		return NewErrorFailed(err)
	}
	log.WithFields(logrus.Fields{"swapID": claim.SwapID}).Info("Claim tx sent to Kava: ", res.Hash.String())
	return nil
}

func broadcastErrorIsRetryable(err error) bool {
	var httpClientError *url.Error
	// retry if there's an error in posting the tx
	isRetryable := errors.As(err, &httpClientError)
	return isRetryable
}

// errorCodeIsRetryable returns true for temporary kava chain error codes.
func errorCodeIsRetryable(codespace string, code uint32) bool {
	// errors are organized by codespace, then error code. For example codespace:"sdk" code:5 is an insufficient funds error.
	temporaryErrorCodes := map[string](map[uint32]bool){}
	// Common sdk errors are listed in cosmos-sdk/types/errors
	temporaryErrorCodes[sdkerrors.RootCodespace] = map[uint32]bool{
		sdkerrors.ErrUnauthorized.ABCICode():    true, // returned when sig fails due to incorrect sequence
		sdkerrors.ErrInvalidSequence.ABCICode(): true, // not currently used, but if that changes we want to retry
		sdkerrors.ErrMempoolIsFull.ABCICode():   true,
	}
	return temporaryErrorCodes[codespace][code]
}

func getAccountNumbers(client *KavaClient, fromAddr sdk.AccAddress) (uint64, uint64, error) {
	acc, err := client.GetAccount(fromAddr)
	if err != nil {
		return 0, 0, err
	}
	return acc.GetSequence(), acc.GetAccountNumber(), nil
}

// pollWithBackoff will call the provided function until either:
// it returns true, it returns an error, the timeout passes.
// It will wait initialInterval after the first call, and double each subsequent call.
func pollWithBackoff(timeout, initialInterval time.Duration, pollFunc func() (bool, error)) error {
	const backoffMultiplier = 2
	deadline := time.After(timeout)

	wait := initialInterval
	nextPoll := time.After(0)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("polling timed out after %s", timeout)
		case <-nextPoll:
			shouldStop, err := pollFunc()
			if shouldStop || err != nil {
				return err
			}
			nextPoll = time.After(wait)
			wait = wait * backoffMultiplier
		}
	}
}

// calculateFee calculates the total fee to be paid based on a total gas and gas price.
func calculateFee(gas uint64, gasPrice sdk.DecCoin) authtypes.StdFee {
	var coins sdk.Coins
	if gas > 0 {
		coins = sdk.NewCoins(sdk.NewCoin(
			gasPrice.Denom,
			gasPrice.Amount.MulInt64(int64(gas)).Ceil().TruncateInt(),
		))
	}
	return authtypes.NewStdFee(gas, coins)
}
