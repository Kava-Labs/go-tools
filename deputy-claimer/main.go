package main

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
	sdk "github.com/kava-labs/cosmos-sdk/types"
	authexported "github.com/kava-labs/cosmos-sdk/x/auth/exported"
	authtypes "github.com/kava-labs/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	tmbytes "github.com/kava-labs/tendermint/libs/bytes"
	"golang.org/x/sync/semaphore"
)

const ( // TODO move these to env vars
	kavaRestURL         = "http://localhost:1317"                      // "http://kava3.data.kava.io"
	bnbRPCURL           = "tcp://localhost:26658"                      // "tcp://dataseed1.binance.org:80"
	bnbDeputyAddrString = "bnb1uky3me9ggqypmrsvxk7ur6hqkzq7zmv4ed4ng7" // "bnb1jh7uv2rm6339yue8k4mj9406k3509kr4wt5nxn"
	kavaChainID         = "testing"                                    // "kava-3" // TODO query from node
)

var claimerMnemonics = []string{
	"census museum crew rude tower vapor mule rib weasel faith page cushion rain inherit much cram that blanket occur region track hub zero topple",
	"flavor print loyal canyon expand salmon century field say frequent human dinosaur frame claim bridge affair web way direct win become merry crash frequent",
}

type restResponse struct {
	Height int             `json:"height"`
	Result json.RawMessage `json:"result"`
}
type restPostTxRequest struct {
	Tx   authtypes.StdTx `json:"tx"`
	Mode string          `json:"mode"`
}

func main() {
	for {
		err := RunKava()
		if err != nil {
			log.Println(err)
		}
		time.Sleep(5 * time.Minute)
	}
	// repeat for bnb
}

func RunKava() error {

	// setup kava codec and config
	cdc := kava.MakeCodec()
	kavaConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(kavaConfig)
	kavaConfig.Seal()

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

	fmt.Println("fetched swaps: ", swaps)

	// parse out swap ids, query those txs on bnb, extract random numbers
	bnbDeputyAddr, err := types.AccAddressFromBech32(bnbDeputyAddrString)
	if err != nil {
		return err
	}
	bnbClient := bnbRpc.NewRPCClient(bnbRPCURL, types.ProdNetwork)
	var rndNums []tmbytes.HexBytes
	for _, s := range swaps { // TODO could be concurrent
		bID := bnbmsg.CalculateSwapID(s.RandomNumberHash, bnbDeputyAddr, s.Sender.String())
		bnbSwap, err := bnbClient.GetSwapByID(bID)
		if err != nil {
			// do nothing, log maybe
			return err
		}
		rndNums = append(rndNums, tmbytes.HexBytes(bnbSwap.RandomNumber))
	}

	fmt.Println("parsed random numbers: ", rndNums)

	// create and submit claim txs, distributing work over several addresses to avoid sequence number problems
	sem := semaphore.NewWeighted(int64(len(claimerMnemonics)))
	ctx := context.TODO()
	errs := make(chan error, len(rndNums))
	for i, r := range rndNums {
		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}
		go func(i int, r tmbytes.HexBytes) {
			defer sem.Release(1)

			// choose private key
			mnemonic := claimerMnemonics[i%len(claimerMnemonics)]
			kavaKeyM, err := kavaKeys.NewMnemonicKeyManager(mnemonic, kava.Bip44CoinType)
			if err != nil {
				errs <- err
				return
			}
			// construct and sign tx
			msg := bep3.NewMsgClaimAtomicSwap(kavaKeyM.GetAddr(), swaps[i].GetSwapID(), r)
			fmt.Println("msg: ", msg)
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
			fmt.Println("account resp: ", string(bz))
			var res restResponse
			cdc.MustUnmarshalJSON(bz, &res)
			var account authexported.Account
			cdc.MustUnmarshalJSON(res.Result, &account)
			fmt.Println("account: ", account)
			signMsg := authtypes.StdSignMsg{
				ChainID:       kavaChainID,
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
			cdc.UnmarshalBinaryLengthPrefixed(txBz, &tx) // TODO
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
			fmt.Println("post req: ", string(reqBz))
			resp, err = http.Post(kavaRestURL+"/txs", "application/json", bytes.NewBuffer(reqBz))
			if err != nil {
				errs <- err
				return
			}
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(body))
			// TODO unmarshal and check error code was 0
			// TODO wait until tx in block, rather than just waiting
			time.Sleep(5 * time.Second)
		}(i, r)
	}

	// wait for all go routines to finish
	if err := sem.Acquire(ctx, int64(len(claimerMnemonics))); err != nil {
		return err
	}
	// TODO look for "proper" way of handling errors from many goroutines
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
