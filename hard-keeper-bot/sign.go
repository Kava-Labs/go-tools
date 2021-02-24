package main

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
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
		// keep track of all signed inflight txs
		// index is sequence % inflightTxLimit
		inflight := make([]*MsgResponse, s.inflightTxLimit)
		// used for confirming sent txs only
		prevDeliverTxSeq := account.GetSequence()
		// tx sequence of mempool
		checkTxSeq := account.GetSequence()

		for {
			if account.GetSequence() > prevDeliverTxSeq {
				// let caller know txs have been included in blocks
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

			// if max number of inflight messages is reached, wait for account update and check again
			if checkTxSeq-account.GetSequence() >= s.inflightTxLimit {
				account = <-accountState
				continue
			}

			select {
			// if no messages, wait for an account state update and continue from top of loop
			case account = <-accountState:
			case request := <-requests:
				for {
					// recover from an unauthorized error
					if checkTxSeq < account.GetSequence() {
						checkTxSeq = account.GetSequence()
					}

					// check if we already have a message inflight for the current seq number
					response := inflight[checkTxSeq%s.inflightTxLimit]

					if response == nil {
						stdSignMsg := authtypes.StdSignMsg{
							ChainID:       chainID,
							AccountNumber: account.GetAccountNumber(),
							Sequence:      checkTxSeq,
							Fee:           request.Fee,
							Msgs:          request.Msgs,
							Memo:          request.Memo,
						}
						stdTx, err := Sign(s.privKey, stdSignMsg)
						response = &MsgResponse{
							Request: request,
							Tx:      stdTx,
							Err:     err,
						}

						inflight[checkTxSeq%s.inflightTxLimit] = response

						// tx malformed, could not sign
						if response.Err != nil {
							break
						}
					}

					result, err := s.client.BroadcastTxSync(&response.Tx)
					if err != nil {
						response.Err = err
						// node may be down, pause and try again
						time.Sleep(1 * time.Second)
						continue
					}
					response.Result = *result
					response.Err = nil

					// 4: unauthorized
					if result.Code == sdkerrors.ErrUnauthorized.ABCICode() {
						// untracked tx in mempool, or dropped mempool tx
						// wait for state refresh and recover, re-sending from last deliver tx seq
						account = <-accountState
						checkTxSeq = account.GetSequence()
						continue
					}

					// 20: mempool full
					if result.Code == sdkerrors.ErrMempoolIsFull.ABCICode() {
						// wait for state refresh and try again
						account = <-accountState
						continue
					}

					// 0: success, in mempool
					// 19: success, tx already in mempool
					if result.Code == sdkerrors.SuccessABCICode || result.Code == sdkerrors.ErrTxInMempoolCache.ABCICode() {
						checkTxSeq = checkTxSeq + 1
					} else {
						response.Err = fmt.Errorf("message failed to broadcast, unrecoverable error code %d", result.Code)
					}

					// exit loop, msg has been processed
					break
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
