package claimer

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bep3types "github.com/kava-labs/kava/x/bep3/types"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/types"
	"github.com/kava-labs/go-tools/signing"
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

func claimOnBinanceChain(bnbHTTP brpc.Client, claim types.ClaimJob) (string, string, error) {
	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		return "", "", NewErrorFailed(err)
	}

	swap, err := bnbHTTP.GetSwapByID(swapID[:])
	if err != nil {
		if strings.Contains(err.Error(), "zero records") {
			return "", "", NewErrorRetryable(fmt.Errorf("swap %s not found in state", claim.SwapID))
		}
		return "", "", NewErrorFailed(err)
	}
	// return the swap recipient to add to logs to help in debugging
	recipient := swap.To.String()

	status, err := bnbHTTP.Status()
	if err != nil {
		return "", recipient, NewErrorRetryable(err)
	}

	if swap.Status != btypes.Open || status.SyncInfo.LatestBlockHeight >= swap.ExpireHeight {
		return "", recipient, NewErrorFailed(fmt.Errorf("swap %s has status %s and cannot be claimed", claim.SwapID, swap.Status))
	}

	randomNumber, err := hex.DecodeString(claim.RandomNumber)
	if err != nil {
		return "", recipient, NewErrorFailed(err)
	}

	res, err := bnbHTTP.ClaimHTLT(swapID[:], randomNumber[:], brpc.Commit)
	if err != nil {
		return "", recipient, NewErrorFailed(err)
	}

	if res.Code != 0 {
		return "", recipient, NewErrorFailed(errors.New(res.Log))
	}

	return res.Hash.String(), recipient, nil
}

func claimOnKava(config config.KavaConfig, client KavaChainClient, claim types.ClaimJob, privKey cryptotypes.PrivKey) (string, string, error) {
	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		return "", "", NewErrorFailed(err)
	}

	swap, err := client.GetSwapByID(swapID)
	if err != nil {
		return "", "", NewErrorRetryable(fmt.Errorf("failed to fetch swap %v", err))
	}
	// return the swap recipient to add to logs to help in debugging
	recipient := swap.Recipient

	if swap.Status == bep3types.SWAP_STATUS_UNSPECIFIED {
		return "", recipient, NewErrorRetryable(fmt.Errorf("swap %s not found in state", swapID))
	} else if swap.Status == bep3types.SWAP_STATUS_EXPIRED || swap.Status == bep3types.SWAP_STATUS_COMPLETED {
		return "", recipient, NewErrorFailed(fmt.Errorf("swap %s has status %s and cannot be claimed", hex.EncodeToString(swapID), swap.Status))
	}

	fromAddr := sdk.AccAddress(privKey.PubKey().Address())

	randomNumber, err := hex.DecodeString(claim.RandomNumber)
	if err != nil {
		return "", recipient, NewErrorFailed(err)
	}

	msg := bep3types.NewMsgClaimAtomicSwap(fromAddr.String(), swapID, randomNumber)
	if err := msg.ValidateBasic(); err != nil {
		return "", recipient, NewErrorFailed(fmt.Errorf("msg basic validation failed: \n%v", msg))
	}

	encodingConfig := client.GetEncodingCoding()

	txBuilder := encodingConfig.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&msg)
	txBuilder.SetGasLimit(ClaimTxDefaultGas)
	txBuilder.SetFeeAmount(calculateFee(ClaimTxDefaultGas, DefaultGasPrice))

	sequence, accountNumber, err := getAccountNumbers(client, fromAddr)
	if err != nil {
		return "", recipient, NewErrorRetryable(fmt.Errorf("failed to fetch account number %v", err))
	}

	signerData := authsigning.SignerData{
		ChainID:       config.ChainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}

	_, txBytes, err := signing.Sign(encodingConfig.TxConfig, privKey, txBuilder, signerData)
	if err != nil {
		return "", recipient, NewErrorFailed(err)
	}

	request := txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	}

	res, err := client.BroadcastTx(request)
	if err != nil {
		if broadcastErrorIsRetryable(err) {
			return "", recipient, NewErrorRetryable(err)
		}
		return "", recipient, NewErrorFailed(err)
	}
	if res.TxResponse.Code != 0 {
		if errorCodeIsRetryable(res.TxResponse.Codespace, res.TxResponse.Code) {
			return "", recipient, NewErrorRetryable(fmt.Errorf("tx rejected from mempool: %s", res.TxResponse.Logs))
		}
		return "", recipient, NewErrorFailed(fmt.Errorf("tx rejected from mempool: %s", res.TxResponse.Logs))
	}
	err = pollWithBackoff(TxConfirmationTimeout, TxConfirmationPollInterval, func() (bool, error) {
		queryRes, err := client.GetTxConfirmation(res.TxResponse.TxHash)
		if err != nil {
			return false, nil // poll again, it can't find the tx or node is down/slow
		}
		if queryRes.Code != 0 {
			return true, fmt.Errorf("tx rejected from block: %s", queryRes.Logs) // return error, found tx but it didn't work
		}
		return true, nil // return nothing, found successfully confirmed tx
	})
	if err != nil {
		return "", recipient, NewErrorFailed(err)
	}

	return res.TxResponse.TxHash, recipient, nil
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

func getAccountNumbers(client KavaChainClient, fromAddr sdk.AccAddress) (uint64, uint64, error) {
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
func calculateFee(gas uint64, gasPrice sdk.DecCoin) sdk.Coins {
	var coins sdk.Coins
	if gas > 0 {
		coins = sdk.NewCoins(sdk.NewCoin(
			gasPrice.Denom,
			gasPrice.Amount.MulInt64(int64(gas)).Ceil().TruncateInt(),
		))
	}
	return coins
}
