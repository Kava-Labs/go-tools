package claim

import (
	"bytes"
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
	sdk "github.com/kava-labs/cosmos-sdk/types"
	authexported "github.com/kava-labs/cosmos-sdk/x/auth/exported"
	authtypes "github.com/kava-labs/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	tmbytes "github.com/kava-labs/tendermint/libs/bytes"
	"golang.org/x/sync/semaphore"
)

type restResponse struct {
	Height int             `json:"height"`
	Result json.RawMessage `json:"result"`
}
type restPostTxRequest struct {
	Tx   authtypes.StdTx `json:"tx"`
	Mode string          `json:"mode"`
}

func RunKava(kavaRestURL, bnbRPCURL string, bnbDeputyAddrString string, mnemonics []string) error {

	// setup kava codec
	cdc := kava.MakeCodec()

	// query kava swaps (via rest)
	resp, err := http.Get(kavaRestURL + "/bep3/swaps?direction=outgoing&status=open&limit=1000")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var res restResponse
	cdc.MustUnmarshalJSON(bz, &res)
	var swaps bep3.AtomicSwaps
	cdc.MustUnmarshalJSON(res.Result, &swaps)
	// filter out swaps to
	var filteredSwaps bep3.AtomicSwaps
	for _, s := range swaps {
		if time.Unix(s.Timestamp, 0).Add(10 * time.Minute).Before(time.Now()) {
			filteredSwaps = append(filteredSwaps, s)
		}
	}

	// parse out swap ids, query those txs on bnb, extract random numbers
	bnbDeputyAddr, err := types.AccAddressFromBech32(bnbDeputyAddrString)
	if err != nil {
		return err
	}
	bnbClient := bnbRpc.NewRPCClient(bnbRPCURL, types.ProdNetwork)
	var rndNums []tmbytes.HexBytes
	for _, s := range filteredSwaps { // TODO could be concurrent
		bID := bnbmsg.CalculateSwapID(s.RandomNumberHash, bnbDeputyAddr, s.Sender.String())
		bnbSwap, err := bnbClient.GetSwapByID(bID)
		if err != nil {
			return err
		}
		rndNums = append(rndNums, tmbytes.HexBytes(bnbSwap.RandomNumber))
	}
	log.Printf("found %d claimable kava HTLTs\n", len(rndNums))

	// Get the chain id
	infoResp, err := http.Get(kavaRestURL + "/node_info")
	if err != nil {
		return err
	}
	defer infoResp.Body.Close()
	infoBz, err := ioutil.ReadAll(infoResp.Body)
	if err != nil {
		return err
	}
	var nodeInfo rpc.NodeInfoResponse
	cdc.MustUnmarshalJSON(infoBz, &nodeInfo)
	chainID := nodeInfo.Network

	// create and submit claim txs, distributing work over several addresses to avoid sequence number problems
	sem := semaphore.NewWeighted(int64(len(mnemonics)))
	ctx := context.TODO()
	errs := make(chan error, len(rndNums))
	for i, r := range rndNums {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		go func(i int, r tmbytes.HexBytes) {
			log.Printf("sending claim for kava swap id %s", filteredSwaps[i].GetSwapID())
			defer sem.Release(1)

			// choose private key
			mnemonic := mnemonics[i%len(mnemonics)]
			kavaKeyM, err := kavaKeys.NewMnemonicKeyManager(mnemonic, kava.Bip44CoinType)
			if err != nil {
				errs <- err
				return
			}
			// construct and sign tx
			msg := bep3.NewMsgClaimAtomicSwap(kavaKeyM.GetAddr(), filteredSwaps[i].GetSwapID(), r)
			resp, err := http.Get(kavaRestURL + "/auth/accounts/" + kavaKeyM.GetAddr().String()) // TODO construct urls properly
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			bz, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				errs <- err
				return
			}
			var res restResponse
			cdc.MustUnmarshalJSON(bz, &res)
			var account authexported.Account
			cdc.MustUnmarshalJSON(res.Result, &account)
			signMsg := authtypes.StdSignMsg{
				ChainID:       chainID,
				AccountNumber: account.GetAccountNumber(),
				Sequence:      account.GetSequence(),
				Fee:           authtypes.NewStdFee(250000, nil),
				Msgs:          []sdk.Msg{msg},
				Memo:          "",
			}
			txBz, err := kavaKeyM.Sign(signMsg, cdc)
			if err != nil {
				errs <- err
				return
			}
			var tx authtypes.StdTx
			err = cdc.UnmarshalBinaryLengthPrefixed(txBz, &tx) // TODO
			if err != nil {
				errs <- err
				return
			}
			// broadcast tx to chain
			req := restPostTxRequest{
				Tx:   tx,
				Mode: "block",
			}
			reqBz, err := cdc.MarshalJSON(req)
			if err != nil {
				errs <- err
				return
			}
			resp, err = http.Post(kavaRestURL+"/txs", "application/json", bytes.NewBuffer(reqBz))
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			// TODO unmarshal body and check error code was 0
			// body, _ := ioutil.ReadAll(resp.Body)

			time.Sleep(7 * time.Second) // TODO wait until tx in block, rather than just sleeping
		}(i, r)
	}

	// wait for all go routines to finish
	if err := sem.Acquire(ctx, int64(len(mnemonics))); err != nil {
		return err
	}
	// TODO look for "proper" way of handling errors from many goroutines (sync/errgroup?)
	var finalErr string
	close(errs)
	for e := range errs {
		finalErr += e.Error()
	}
	if finalErr != "" {
		return fmt.Errorf("sending claims produced some errors: %s", finalErr)
	}
	return nil
}
