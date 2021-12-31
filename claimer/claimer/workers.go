package claimer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

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

func claimOnKava(config config.KavaConfig, client *KavaClient, claim server.ClaimJob, privKey cryptotypes.PrivKey) error {
	swapID, err := hex.DecodeString(claim.SwapID)
	if err != nil {
		return NewErrorFailed(err)
	}

	swap, err := client.GetSwapByID(context.Background(), swapID)
	if err != nil {
		return NewErrorRetryable(err)
	}
	if swap.Status == bep3types.SWAP_STATUS_UNSPECIFIED {
		return NewErrorRetryable(fmt.Errorf("swap %s not found in state", swapID))
	} else if swap.Status == bep3types.SWAP_STATUS_EXPIRED || swap.Status == bep3types.SWAP_STATUS_COMPLETED {
		return NewErrorFailed(fmt.Errorf("swap %s has status %s and cannot be claimed", hex.EncodeToString(swapID), swap.Status))
	}

	fromAddr := sdk.AccAddress(privKey.PubKey().Address())

	randomNumber, err := hex.DecodeString(claim.RandomNumber)
	if err != nil {
		return NewErrorFailed(err)
	}

	msg := bep3types.NewMsgClaimAtomicSwap(fromAddr.String(), swapID, randomNumber)
	if err := msg.ValidateBasic(); err != nil {
		return NewErrorFailed(fmt.Errorf("msg basic validation failed: \n%v", msg))
	}

	txBuilder := client.encodingConfig.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(&msg)
	txBuilder.SetGasLimit(ClaimTxDefaultGas)
	txBuilder.SetFeeAmount(calculateFee(ClaimTxDefaultGas, DefaultGasPrice))

	sequence, accountNumber, err := getAccountNumbers(client, fromAddr)
	if err != nil {
		return NewErrorFailed(err)
	}

	signatureData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}
	sigV2 := signing.SignatureV2{
		PubKey:   privKey.PubKey(),
		Data:     &signatureData,
		Sequence: sequence,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return NewErrorFailed(err)
	}

	signerData := authsigning.SignerData{
		ChainID:       config.ChainID,
		AccountNumber: accountNumber,
		Sequence:      sequence,
	}
	signBytes, err := client.encodingConfig.TxConfig.SignModeHandler().GetSignBytes(signing.SignMode_SIGN_MODE_DIRECT, signerData, txBuilder.GetTx())
	if err != nil {
		return NewErrorFailed(err)
	}
	signature, err := privKey.Sign(signBytes)
	if err != nil {
		return NewErrorFailed(err)
	}

	sigV2.Data = &signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: signature,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return NewErrorFailed(err)
	}

	txBytes, err := client.encodingConfig.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return NewErrorFailed(err)
	}

	res, err := client.BroadcastTxSync(context.Background(), txBytes)
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
		queryRes, err := client.GetTxConfirmation(context.Background(), res.Hash)
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
	acc, err := client.GetAccount(context.Background(), fromAddr)
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
