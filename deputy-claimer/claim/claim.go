package claim

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"
	bnbmsg "github.com/binance-chain/go-sdk/types/msg"
	"github.com/kava-labs/cosmos-sdk/client/rpc"
	"github.com/kava-labs/cosmos-sdk/codec"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	authexported "github.com/kava-labs/cosmos-sdk/x/auth/exported"
	authtypes "github.com/kava-labs/cosmos-sdk/x/auth/types"
	kavaRpc "github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	tmbytes "github.com/kava-labs/tendermint/libs/bytes"
	tmRPCTypes "github.com/kava-labs/tendermint/rpc/core/types"
	tmtypes "github.com/kava-labs/tendermint/types"
	"golang.org/x/sync/semaphore"
)

func RunKava(kavaRestURL, kavaRPCURL, bnbRPCURL string, bnbDeputyAddrString string, mnemonics []string) error {

	// setup
	cdc := kava.MakeCodec()
	kavaClient := NewKavaChainClient(kavaRestURL, kavaRPCURL, cdc)
	bnbClient := bnbRpc.NewRPCClient(bnbRPCURL, types.ProdNetwork)

	bnbDeputyAddr, err := types.AccAddressFromBech32(bnbDeputyAddrString)
	if err != nil {
		return err
	}

	claimableSwaps, err := getClaimableKavaSwaps(kavaClient, bnbClient, bnbDeputyAddr)
	if err != nil {
		return err
	}
	log.Printf("found %d claimable kava HTLTs\n", len(claimableSwaps))

	// create and submit claim txs, distributing work over several addresses to avoid sequence number problems
	sem := semaphore.NewWeighted(int64(len(mnemonics)))
	ctx := context.TODO()
	errs := make(chan error, len(claimableSwaps))
	for i, swap := range claimableSwaps {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		go func(i int, swap claimableSwap) {
			log.Printf("sending claim for kava swap id %s", swap.swapID)
			defer sem.Release(1)

			// FIXME semaphore releases not synced with choosing mnemonics, don't know which mnemonic is free
			txHash, err := constructAndSendClaim(kavaClient, mnemonics[i%len(mnemonics)], swap.swapID, swap.randomNumber)
			if err != nil {
				errs <- err
				return
			}
			err = waitWithTimeoutForTxSuccess(kavaClient, 15*time.Second, txHash)
			if err != nil {
				errs <- err
				return
			}
		}(i, swap)
	}

	// wait for all go routines to finish
	if err := sem.Acquire(ctx, int64(len(mnemonics))); err != nil {
		return err
	}
	// report any errors
	var concatenatedErrs string
	close(errs)
	for e := range errs {
		concatenatedErrs += e.Error()
		concatenatedErrs += "\n"
	}
	if concatenatedErrs != "" {
		return fmt.Errorf("sending claims produced some errors: \n%s", concatenatedErrs)
	}
	return nil
}

type claimableSwap struct {
	swapID       tmbytes.HexBytes
	randomNumber tmbytes.HexBytes
}

func getClaimableKavaSwaps(kavaClient kavaChainClient, bnbClient *bnbRpc.HTTP, bnbDeputyAddr types.AccAddress) ([]claimableSwap, error) {
	swaps, err := kavaClient.getOpenSwaps()
	if err != nil {
		return nil, err
	}

	// filter out new swaps
	var filteredSwaps bep3.AtomicSwaps
	for _, s := range swaps {
		if time.Unix(s.Timestamp, 0).Add(10 * time.Minute).Before(time.Now()) {
			filteredSwaps = append(filteredSwaps, s)
		}
	}

	// parse out swap ids, query those txs on bnb, extract random numbers
	var claimableSwaps []claimableSwap
	for _, s := range filteredSwaps {
		bID := bnbmsg.CalculateSwapID(s.RandomNumberHash, bnbDeputyAddr, s.Sender.String())
		bnbSwap, err := bnbClient.GetSwapByID(bID)
		if err != nil {
			return nil, err // TODO should probably just continue rather than stopping, or parse not found error
		}
		// check the bnb swap status is closed and has random number - ie it has been claimed
		if len(bnbSwap.RandomNumber) != 0 {
			claimableSwaps = append(
				claimableSwaps,
				claimableSwap{
					swapID:       s.GetSwapID(),
					randomNumber: tmbytes.HexBytes(bnbSwap.RandomNumber),
				})
		}
	}
	return claimableSwaps, nil
}

func constructAndSendClaim(kavaClient kavaChainClient, mnemonic string, swapID, randNum tmbytes.HexBytes) ([]byte, error) {
	kavaKeyM, err := kavaKeys.NewMnemonicKeyManager(mnemonic, kava.Bip44CoinType)
	if err != nil {
		return nil, err
	}
	// construct and sign tx
	msg := bep3.NewMsgClaimAtomicSwap(kavaKeyM.GetAddr(), swapID, randNum)
	chainID, err := kavaClient.getChainID()
	if err != nil {
		return nil, err
	}
	account, err := kavaClient.getAccount(kavaKeyM.GetAddr())
	if err != nil {
		return nil, err
	}
	signMsg := authtypes.StdSignMsg{
		ChainID:       chainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
		Fee:           authtypes.NewStdFee(250000, nil),
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}
	txBz, err := kavaKeyM.Sign(signMsg, kavaClient.codec)
	if err != nil {
		return nil, err
	}
	// broadcast tx to mempool
	if err = kavaClient.broadcastTx(txBz); err != nil {
		return nil, err
	}
	return tmtypes.Tx(txBz).Hash(), nil
}
func waitWithTimeoutForTxSuccess(kavaClient kavaChainClient, timeout time.Duration, txHash []byte) error {
	endTime := time.Now().Add(timeout)
	for {
		res, err := kavaClient.getTxConfirmation(txHash)
		if err != nil {
			// TODO parse error to see if the was found or not
			if time.Now().After(endTime) {
				return fmt.Errorf("timeout reached")
			} else {
				time.Sleep(1 * time.Second)
				continue
			}
		}
		if res.TxResult.Code != 0 {
			return fmt.Errorf("tx rejected from chain: %s", res.TxResult.Log)
		}
		return nil
	}
}

type kavaChainClient struct {
	restURL, rpcURL string
	codec           *codec.Codec
	kavaSDKClient   *kavaRpc.KavaClient
}

func NewKavaChainClient(restURL, rpcURL string, cdc *codec.Codec) kavaChainClient {
	// use a fake mnemonic as we're not using the kava client for signing
	dummyMnemonic := "adult stem bus people vast riot eager faith sponsor unlock hold lion sport drop eyebrow loud angry couch panic east three credit grain talk"
	return kavaChainClient{
		restURL:       restURL,
		codec:         cdc,
		kavaSDKClient: kavaRpc.NewKavaClient(cdc, dummyMnemonic, kava.Bip44CoinType, rpcURL, kavaRpc.ProdNetwork), // TODO what is network type for?
	}
}

type restResponse struct {
	Height int             `json:"height"`
	Result json.RawMessage `json:"result"`
}

func (kc kavaChainClient) getOpenSwaps() (bep3.AtomicSwaps, error) {
	resp, err := http.Get(kc.restURL + "/bep3/swaps?direction=outgoing&status=open&limit=1000")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res restResponse
	kc.codec.MustUnmarshalJSON(bz, &res)
	var swaps bep3.AtomicSwaps
	kc.codec.MustUnmarshalJSON(res.Result, &swaps)
	return swaps, nil
}

func (kc kavaChainClient) getAccount(address sdk.AccAddress) (authexported.Account, error) {
	resp, err := http.Get(kc.restURL + "/auth/accounts/" + address.String()) // TODO construct urls properly
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res restResponse
	kc.codec.MustUnmarshalJSON(bz, &res)
	var account authexported.Account
	kc.codec.MustUnmarshalJSON(res.Result, &account)
	return account, nil
}

func (kc kavaChainClient) getTxConfirmation(txHash []byte) (*tmRPCTypes.ResultTx, error) {
	return kc.kavaSDKClient.HTTP.Tx(txHash, false)
}

func (kc kavaChainClient) broadcastTx(tx tmtypes.Tx) error {
	res, err := kc.kavaSDKClient.BroadcastTxSync(tx)
	if err != nil {
		return err
	}
	if res.Code != 0 { // tx failed to be submitted to the mempool
		return fmt.Errorf("transaction failed to get into mempool: %s", res.Log)
	}
	return nil
}

func (kc kavaChainClient) getChainID() (string, error) {
	infoResp, err := http.Get(kc.restURL + "/node_info")
	if err != nil {
		return "", err
	}
	defer infoResp.Body.Close()
	infoBz, err := ioutil.ReadAll(infoResp.Body)
	if err != nil {
		return "", err
	}
	var nodeInfo rpc.NodeInfoResponse
	kc.codec.MustUnmarshalJSON(infoBz, &nodeInfo)
	return nodeInfo.Network, nil
}
