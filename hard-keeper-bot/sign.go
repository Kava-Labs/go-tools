package main

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmmempool "github.com/tendermint/tendermint/mempool"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmjsonrpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"
)

type MsgRequest struct {
	Msgs []sdk.Msg
	Fee  authtypes.StdFee
	Memo string
}

type MsgResponse struct {
	Request MsgRequest
	Tx      authtypes.StdTx
	Result  ctypes.ResultBroadcastTx
	Err     error
}

var tmMempoolFull = regexp.MustCompile("mempool is full")

// internal result for inner loop logic
type broadcastTxResult int

const (
	txOK broadcastTxResult = iota
	txFailed
	txRetry
	txUnauthorized
)

// Signer broadcasts msgs to a single kava node
type Signer struct {
	client          BroadcastClient
	privKey         tmcrypto.PrivKey
	inflightTxLimit uint64
}

func NewSigner(client BroadcastClient, privKey tmcrypto.PrivKey, inflightTxLimit uint64) *Signer {
	return &Signer{
		client:          client,
		privKey:         privKey,
		inflightTxLimit: inflightTxLimit,
	}
}

func (s *Signer) Run(requests <-chan MsgRequest, responses chan<- MsgResponse) error {
	chainID, err := s.client.GetChainID()
	if err != nil {
		return err
	}
	// poll account state in it's own goroutine
	// and send status updates to the signing goroutine
	//
	// TODO: instead of polling, we can wait for block
	// websocket events with a fallback to polling
	accountState := make(chan authtypes.BaseAccount)
	go func() {
		for {
			account, err := s.client.GetAccount(GetAccAddress(s.privKey))
			if err == nil {
				accountState <- *account
			}
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		// wait until account is loaded to start signing
		account := <-accountState
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
			// if the currentRequest has been been successfully broadcast
			//
			// if an unauthorized error occurs, a tx in the mempool was dropped (or mempool flushed, node restart, etc)
			// and broadcastTxSeq is reset to account.GetSequence() in order to refil the mempool and ensure
			// checkTxSeq is valid
			//
			// if an authorized error occurs due to another process signing messages on behalf of the same
			// address, then broadcastTxSeq will continually be reset until that sequence is delivered to a block
			//
			// this results in the message we signed with the same sequence being skipped as well as
			// draining our inflight messages to 0.
			//
			// On deployments, a similar event will occur. we will continually broadcast until
			// all of the previous transactions are processed and out of the mempool.
			//
			// it's possible to increase the checkTx (up to the inflight limit) until met with a successful broadcast,
			// to fill the mempool faster, but this feature is umimplemented and would be best enabled only once
			// on startup.  An authorized error during normal operation would be difficult or impossible to tell apart
			// from a dropped mempool tx (without further improving mempool queries).  Options such as persisting inflight
			// state out of process may be better.
			inflightLimitReached := checkTxSeq-account.GetSequence() >= s.inflightTxLimit

			// if we are still processing a request or the inflight limit is reached
			// then block until the next account update without accepting new requests
			if currentRequest != nil || inflightLimitReached {
				account = <-accountState
			} else {
				// block on state update or new requests
				select {
				case account = <-accountState:
				case request := <-requests:
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
			if currentRequest == nil {
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
		BROADCAST_LOOP:
			for broadcastTxSeq <= lastRequestTxSeq {

				// we have a new request that has not been successfully broadcasted
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

					stdSignMsg := authtypes.StdSignMsg{
						ChainID:       chainID,
						AccountNumber: account.GetAccountNumber(),
						Sequence:      broadcastTxSeq,
						Fee:           currentRequest.Fee,
						Msgs:          currentRequest.Msgs,
						Memo:          currentRequest.Memo,
					}

					stdTx, err := Sign(s.privKey, stdSignMsg)

					response = &MsgResponse{
						Request: *currentRequest,
						Tx:      stdTx,
						Err:     err,
					}

					// could not sign the currentRequest
					if response.Err != nil {
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
				rpcResult, err := s.client.BroadcastTxSync(&response.Tx)

				// set to determine action at the end of loop
				// default is OK
				txResult := txOK

				// determine action to take when err (and no rpcResult)
				if err != nil {
					var rpcError *tmjsonrpctypes.RPCError

					// tendermint rpc error
					if errors.As(err, &rpcError) {
						if rpcError.Data == tmmempool.ErrTxInCache.Error() {
							txResult = txOK
						} else if tmMempoolFull.MatchString(rpcError.Data) {
							txResult = txRetry
						} else {
							// ErrTxTooLarge or ErrPreCheck - not recoverable
							// other RPC Errors like parsing, etc
							response.Err = rpcError
							txResult = txFailed
						}
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
					response.Result = *rpcResult

					// determine action to take based on rpc result
					switch rpcResult.Code {
					// 0: success, in mempool
					case sdkerrors.SuccessABCICode:
						txResult = txOK
					// 4: unauthorized
					case sdkerrors.ErrUnauthorized.ABCICode():
						txResult = txUnauthorized
					// 19: success, tx already in mempool
					case sdkerrors.ErrTxInMempoolCache.ABCICode():
						txResult = txOK
					// 20: mempool full
					case sdkerrors.ErrMempoolIsFull.ABCICode():
						txResult = txRetry
					default:
						response.Err = fmt.Errorf("message failed to broadcast, unrecoverable error code %d", rpcResult.Code)
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
					// do not store the request as inflight (it's not in the mempool)
					inflight[broadcastTxSeq%s.inflightTxLimit] = nil

					// clear current request if it failed
					if sendingCurrentRequest {
						currentRequest = nil
					}

					// immediatley response to channel
					responses <- *response
					// go to next request
					broadcastTxSeq++
				case txRetry:
					break BROADCAST_LOOP
				case txUnauthorized:
					broadcastTxSeq = account.GetSequence()
					break BROADCAST_LOOP
				}
			}
		}
	}()

	return nil
}

func GetAccAddress(privKey tmcrypto.PrivKey) sdk.AccAddress {
	return privKey.PubKey().Address().Bytes()
}

func Sign(privKey tmcrypto.PrivKey, signMsg authtypes.StdSignMsg) (authtypes.StdTx, error) {
	sigBytes, err := privKey.Sign(signMsg.Bytes())
	if err != nil {
		return authtypes.StdTx{}, err
	}

	sig := authtypes.StdSignature{
		PubKey:    privKey.PubKey(),
		Signature: sigBytes,
	}

	tx := authtypes.NewStdTx(
		signMsg.Msgs,
		signMsg.Fee,
		[]authtypes.StdSignature{sig},
		signMsg.Memo,
	)

	return tx, nil
}
