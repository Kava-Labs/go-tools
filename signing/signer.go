package signing

import (
	"context"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/rs/zerolog"
	tmmempool "github.com/tendermint/tendermint/mempool"
)

// MsgRequest contains the signing request
type MsgRequest struct {
	Msgs      []sdk.Msg   // Messages to be included in the transaction
	GasLimit  uint64      // Gas limit for the transaction
	FeeAmount sdk.Coins   // Fees for the transaction
	Memo      string      // Memo field for the transaction
	Data      interface{} // Arbitrary data for matching responses with requests
}

// MsgResponse contains the signing response
type MsgResponse struct {
	Request MsgRequest     // The original request that was signed
	Tx      authsigning.Tx // The signed transaction
	TxBytes []byte         // The raw bytes of the signed transaction
	Result  sdk.TxResponse // The result of broadcasting the transaction
	Err     error          // Any error encountered during signing or broadcasting
}

// internal result for inner loop logic
type broadcastTxResult int

const (
	txOK            broadcastTxResult = iota // Transaction successfully broadcast and in the mempool
	txFailed                                 // Transaction failed and is not recoverable
	txRetry                                  // Transaction failed but can be retried
	txResetSequence                          // Transaction sequence is invalid, needs to be reset
)

// EncodingConfig defines the necessary methods for encoding and decoding transactions to be able to reuse the signer
// for cosmos-sdk chains other than kava
type EncodingConfig interface {
	InterfaceRegistry() types.InterfaceRegistry // Returns the interface registry
	Marshaler() codec.Codec                     // Returns the codec for marshaling
	TxConfig() client.TxConfig                  // Returns the transaction configuration
	Amino() *codec.LegacyAmino                  // Returns the legacy Amino codec
}

// Signer broadcasts msgs to a single kava node
type Signer struct {
	chainID         string
	encodingConfig  EncodingConfig
	authClient      authtypes.QueryClient
	txClient        txtypes.ServiceClient
	privKey         cryptotypes.PrivKey
	inflightTxLimit uint64
	logger          zerolog.Logger
	accStatus       error
}

// NewSigner creates a new Signer instance
func NewSigner(
	chainID string,
	encodingConfig EncodingConfig,
	authClient authtypes.QueryClient,
	txClient txtypes.ServiceClient,
	privKey cryptotypes.PrivKey,
	inflightTxLimit uint64,
	logger zerolog.Logger,
) (*Signer, error) {
	if inflightTxLimit == 0 {
		return nil, fmt.Errorf("inflightTxLimit cannot be zero")
	}

	return &Signer{
		chainID:         chainID,
		encodingConfig:  encodingConfig,
		authClient:      authClient,
		txClient:        txClient,
		privKey:         privKey,
		inflightTxLimit: inflightTxLimit,
		logger:          logger,
		accStatus:       nil,
	}, nil
}

// GetAccountError returns the error encountered when querying the signing account
func (s *Signer) GetAccountError() error {
	return s.accStatus
}

func (s *Signer) setAccountError(err error) {
	s.accStatus = err
}

func (s *Signer) clearAccountError() {
	s.accStatus = nil
}

// pollAccountState periodically polls the account state, retrying on errors
func (s *Signer) pollAccountState(ctx context.Context, retryInterval, pollInterval time.Duration) <-chan authtypes.AccountI {
	accountState := make(chan authtypes.AccountI)

	go func() {
		defer close(accountState) // Close channel when goroutine exits
		for {
			select {
			case <-ctx.Done():
				s.logger.Info().Msg("Stopping pollAccountState goroutine")
				return
			default:
				account, err := s.getAccountState(ctx)
				if err != nil {
					s.setAccountError(err)
					s.logger.Error().
						Err(err).
						Dur("retryInterval", retryInterval).
						Msg("trying again with delay")
					select {
					case <-ctx.Done():
						return
					case <-time.After(retryInterval):
						continue
					}
				}
				s.clearAccountError()
				accountState <- account
				select {
				case <-ctx.Done():
					return
				case <-time.After(pollInterval):
				}
			}
		}
	}()

	return accountState
}

// getAccountState queries the account state using the private key
func (s *Signer) getAccountState(ctx context.Context) (authtypes.AccountI, error) {
	accAddr := GetAccAddress(s.privKey)
	response, err := s.authClient.Account(ctx, &authtypes.QueryAccountRequest{
		Address: accAddr.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to query signed account address[%s]: %w", accAddr.String(), err)
	}

	var account authtypes.AccountI
	if err = s.encodingConfig.InterfaceRegistry().UnpackAny(response.Account, &account); err != nil {
		return nil, fmt.Errorf("unable to unpack signed account address[%s]: %w", accAddr.String(), err)
	}

	return account, nil
}

// Run starts the signer, processing incoming requests and broadcasting transactions
//
//	The broadcast loop:
//		Ensure Transaction is in Mempool:
//			The loop continuously attempts to broadcast transactions until they are successfully placed into the node's
//			mempool. This involves handling various errors, such as connectivity issues or mempool capacity errors,
//			and retrying as necessary.
//		Handle Sequence Errors:
//			If there are sequence-related errors, the loop resets the sequence to try placing transactions again,
//			ensuring that the mempool is filled correctly.
//		Respond to Requests:
//			Once a transaction is successfully in the mempool, the loop stores the response and
//			processes the next request.
func (s *Signer) Run(ctx context.Context, requests <-chan MsgRequest) (<-chan MsgResponse, error) {
	// poll account state in it's own goroutine
	// and send status updates to the signing goroutine
	//
	// TODO: instead of polling, we can wait for block
	// websocket events with a fallback to polling
	accountState := s.pollAccountState(ctx, 10*time.Second, 1*time.Second)

	responses := make(chan MsgResponse)
	go func() {
		defer close(responses)
		// wait until account is loaded to start signing
		account, ok := <-accountState
		if !ok {
			s.logger.Error().Msg("Failed to load account state")
			return
		}
		// store current request waiting to be broadcasted
		var currentRequest *MsgRequest
		// keep track of all successfully broadcasted txs
		// index is sequence % inflightTxLimit
		inflight := make([]*MsgResponse, s.inflightTxLimit)
		// used for confirming sent txs only
		prevDeliverTxSeq := account.GetSequence()
		// tx sequence of already signed messages
		checkTxSeq := account.GetSequence()
		// tx sequence of broadcast queue, is reset upon
		// unauthorized errors to recheck/refill mempool
		broadcastTxSeq := account.GetSequence()

		for {
			// the inflight limit includes the current request
			//
			// account.GetSequence() represents the first tx in the mempool (at the last known state)
			// or the next sequence to sign with if checkTxSeq == account.GetSequence() (zero msgs in flight)
			//
			// checkTxSeq always represents the next available mempool sequence to sign with
			//
			// if currentRequest is nil, then it will be used for the next request received
			// if currentRequest is not nil, then checkTxSeq will be used to sign that request
			//
			// therefore, assuming no errors, broadcastTxSeq will be checkTxSeq-1 or checkTxSeq, dependent on
			// if the currentRequest has been successfully broadcast
			//
			// if an unauthorized error occurs, a tx in the mempool was dropped (or mempool flushed, node restart, etc.)
			// and broadcastTxSeq is reset to account.GetSequence() in order to refill the mempool and ensure
			// checkTxSeq is valid
			//
			// if an authorized error occurs due to another process signing messages on behalf of the same
			// address, then broadcastTxSeq will continually be reset until that sequence is delivered to a block
			//
			// this results in the message we signed with the same sequence being skipped as well as
			// draining our inflight messages to 0.
			//
			// On deployments, a similar event will occur. we will continually broadcast until
			// all the previous transactions are processed and out of the mempool.
			//
			// it's possible to increase the checkTx (up to the inflight limit) until met with a successful broadcast,
			// to fill the mempool faster, but this feature is unimplemented and would be best enabled only once
			// on startup.  An authorized error during normal operation would be difficult or impossible to tell apart
			// from a dropped mempool tx (without further improving mempool queries).  Options such as persisting inflight
			// state out of process may be better.
			inflightLimitReached := checkTxSeq-account.GetSequence() >= s.inflightTxLimit

			// if we are still processing a request or the inflight limit is reached
			// then block until the next account update without accepting new requests
			if currentRequest != nil || inflightLimitReached {
				account, ok = <-accountState
				if !ok {
					s.logger.Error().Msg("Account state channel closed unexpectedly")
					return
				}
			} else {
				// block on state update or new requests
				select {
				case <-ctx.Done():
					s.logger.Info().Msg("Stopping Run goroutine")
					return
				case account = <-accountState:
				case request, k := <-requests:
					if !k {
						s.logger.Info().Msg("Request channel closed")
						return
					}
					currentRequest = &request
				}
			}

			// send delivered (included in block) responses to caller
			if account.GetSequence() > prevDeliverTxSeq {
				for i := prevDeliverTxSeq; i < account.GetSequence(); i++ {
					response := inflight[i%s.inflightTxLimit]
					// sequences may be skipped due to errors
					if response != nil {
						responses <- *response
					}
					// clear to prevent duplicate confirmations on errors
					inflight[i%s.inflightTxLimit] = nil
				}
				prevDeliverTxSeq = account.GetSequence()
			}

			// recover from errors due to untracked messages in mempool
			// this will happen on deploys, or if another process
			// signs a tx using the same address
			if checkTxSeq < account.GetSequence() {
				checkTxSeq = account.GetSequence()
			}

			// if currentRequest then lastRequestTxSeq == checkTxSeq
			// if not currentRequest then lastRequestTxSeq == checkTxSeq - 1
			lastRequestTxSeq := checkTxSeq
			if currentRequest == nil && lastRequestTxSeq > 0 {
				lastRequestTxSeq--
			}
			// reset broadcast seq if iterated over last request seq
			// we always want to broadcast the current or last request
			// to heartbeat the mempool
			if broadcastTxSeq > lastRequestTxSeq {
				broadcastTxSeq = lastRequestTxSeq
			}

			// loop serves three purposes
			//   - recover from dropped txs (broadcastTxSeq < lastRequestTxSeq)
			//   - send new requests (currentRequest is set)
			//   - send mempool heartbeat (currentRequest is nil)
		BroadcastLoop:
			for broadcastTxSeq <= lastRequestTxSeq {

				// we have a new request that has not been successfully broadcast
				// and are at the last broadcastTxSeq (broadcastTxSeq == checkTxSeq in this case)
				sendingCurrentRequest := broadcastTxSeq == lastRequestTxSeq && currentRequest != nil

				// check if we have a previous response to check/retry/send for the broadcastTxSeq
				response := inflight[broadcastTxSeq%s.inflightTxLimit]

				// no response -- either checkTxSeq was skipped (untracked mempool tx), or
				// we are signing a new transactions (currentRequest is not nil)
				if response == nil {
					// nothing to do if no response to retry and not sending a current request
					if !sendingCurrentRequest {
						// move onto next broadcastTxSeq or exit loop
						broadcastTxSeq++
						continue
					}
					txBuilder := s.encodingConfig.TxConfig().NewTxBuilder()
					txBuilder.SetMsgs(currentRequest.Msgs...)
					txBuilder.SetGasLimit(currentRequest.GasLimit)
					txBuilder.SetFeeAmount(currentRequest.FeeAmount)

					signerData := authsigning.SignerData{
						ChainID:       s.chainID,
						AccountNumber: account.GetAccountNumber(),
						Sequence:      broadcastTxSeq,
					}

					tx, txBytes, err := Sign(s.encodingConfig.TxConfig(), s.privKey, txBuilder, signerData)

					response = &MsgResponse{
						Request: *currentRequest,
						Tx:      tx,
						TxBytes: txBytes,
						Err:     err,
					}

					// could not sign and encode the currentRequest
					if response.Err != nil {
						s.logger.Error().
							Err(err).
							Uint64("sequence", broadcastTxSeq).
							Interface("tx", txBuilder.GetTx()).
							Msg("failed to sign and encode tx")

						// clear invalid request, since this is non-recoverable
						currentRequest = nil

						// response immediately with error
						responses <- *response

						// exit loop
						broadcastTxSeq++
						continue
					}
				}

				// broadcast tx and get result
				//
				// there are four main types of results
				//
				// OK (tx in mempool, store response - add to inflight txs)
				// Retry (tx not in mempool, but retry - do not change inflight status)
				// Failed (tx not in mempool, not recoverable - clear inflight status, reply to channel)
				// Unauthorized (tx not in mempool - sequence not valid)
				broadcastRequest := txtypes.BroadcastTxRequest{
					TxBytes: response.TxBytes,
					Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
				}
				broadcastResponse, err := s.txClient.BroadcastTx(ctx, &broadcastRequest)

				// set to determine action at the end of loop
				// default is OK
				txResult := txOK

				// determine action to take when err (and no response)
				if err != nil {
					s.logger.Error().
						Err(err).
						Uint64("sequence", broadcastTxSeq).
						Interface("tx", response.Tx).
						Msg("failed to broadcast tx")

					if tmmempool.IsPreCheckError(err) {
						// ErrPreCheck - not recoverable
						response.Err = err
						txResult = txFailed
					} else {
						// could not contact node (POST failed, dns errors, etc)
						// exit loop, wait for another account state update
						// TODO: are there cases here that we will never recover from?
						// should we implement retry limit?
						response.Err = err
						txResult = txRetry
					}
				} else {
					// store rpc result in response
					response.Result = *broadcastResponse.TxResponse

					// determine action to take based on rpc result
					switch response.Result.Code {
					// 0: success, in mempool
					case sdkerrors.SuccessABCICode:
						txResult = txOK
					// 4: unauthorized
					case sdkerrors.ErrUnauthorized.ABCICode():
						txResult = txResetSequence
					// 19: success, tx already in mempool
					case sdkerrors.ErrTxInMempoolCache.ABCICode():
						txResult = txOK
					// 20: mempool full
					case sdkerrors.ErrMempoolIsFull.ABCICode():
						txResult = txRetry
					// 32: wrong sequence
					case sdkerrors.ErrWrongSequence.ABCICode():
						txResult = txResetSequence
					default:
						response.Err = fmt.Errorf("message failed to broadcast, unrecoverable error code %d", response.Result.Code)
						txResult = txFailed
					}
				}

				switch txResult {
				case txOK:
					// clear any errors from previous attempts
					response.Err = nil

					// store for delivery later
					inflight[broadcastTxSeq%s.inflightTxLimit] = response

					// if this is the current/last request, then clear
					// the request and increment the checkTxSeq
					if sendingCurrentRequest {
						currentRequest = nil
						checkTxSeq++
					}

					// go to next request
					broadcastTxSeq++
				case txFailed:
					s.logger.Error().
						Err(response.Err).
						Uint64("sequence", broadcastTxSeq).
						Interface("tx", response.Tx).
						Msg("tx failed")

					// do not store the request as inflight (it's not in the mempool)
					inflight[broadcastTxSeq%s.inflightTxLimit] = nil

					// clear current request if it failed
					if sendingCurrentRequest {
						currentRequest = nil
					}

					// immediately respond to channel
					responses <- *response
					// go to next request
					broadcastTxSeq++
				case txRetry:
					break BroadcastLoop
				case txResetSequence:
					broadcastTxSeq = account.GetSequence()
					break BroadcastLoop
				}
			}
		}
	}()

	return responses, nil
}

// Address returns the address of the Signer
func (s *Signer) Address() sdk.AccAddress {
	return GetAccAddress(s.privKey)
}

// Sign signs a transaction using the provided private key and signer data
func Sign(
	txConfig sdkclient.TxConfig,
	privKey cryptotypes.PrivKey,
	txBuilder sdkclient.TxBuilder,
	signerData authsigning.SignerData,
) (authsigning.Tx, []byte, error) {
	signatureData := signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: nil,
	}
	sigV2 := signing.SignatureV2{
		PubKey:   privKey.PubKey(),
		Data:     &signatureData,
		Sequence: signerData.Sequence,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return txBuilder.GetTx(), nil, err
	}

	signBytes, err := txConfig.SignModeHandler().GetSignBytes(
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		return txBuilder.GetTx(), nil, err
	}
	signature, err := privKey.Sign(signBytes)
	if err != nil {
		return txBuilder.GetTx(), nil, err
	}

	sigV2.Data = &signing.SingleSignatureData{
		SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
		Signature: signature,
	}
	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return txBuilder.GetTx(), nil, err
	}

	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return txBuilder.GetTx(), nil, err
	}

	return txBuilder.GetTx(), txBytes, nil
}

// GetAccAddress returns the account address for a given private key
func GetAccAddress(privKey cryptotypes.PrivKey) sdk.AccAddress {
	return privKey.PubKey().Address().Bytes()
}
