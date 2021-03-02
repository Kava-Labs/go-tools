package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank"
	kava "github.com/kava-labs/kava/app"
	"github.com/tendermint/tendermint/libs/log"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

func main() {
	// create base logger
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	//
	// bootstrap kava chain config
	//
	// sets a global cosmos sdk for bech32 prefix
	//
	// required before loading config
	//
	kavaConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(kavaConfig)
	kava.SetBip44CoinType(kavaConfig)
	kavaConfig.Seal()

	//
	// Load config
	//
	// if config is not valid, exit with fatal error
	//
	config, err := LoadConfig(&EnvLoader{})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.With(
		"rpcUrl", config.KavaRpcUrl,
		"liquidationInterval", config.KavaLiquidationInterval.String(),
	).Info("config loaded")

	//
	// bootstrap rpc http clent
	//
	http, err := rpchttpclient.New(config.KavaRpcUrl, "/websocket")
	if err != nil {
		logger.Error("failed to connect")
		logger.Error(err.Error())
		os.Exit(1)
	}
	http.Logger = logger

	//
	// create codec for messages
	//
	cdc := kava.MakeCodec()

	//
	// Test Message Signing
	//
	broadcastClient := NewRpcBroadcastClient(http, cdc)
	mnemonic := "arrive guide way exit polar print kitchen hair series custom siege afraid shrug crew fashion mind script divorce pattern trust project regular robust safe"
	hdPath := keys.CreateHDPath(0, 0)

	derivedPriv, err := keys.StdDeriveKey(mnemonic, "", hdPath.String(), keys.Secp256k1)
	if err != nil {
		logger.Error("failed to derive key")
		logger.Error(err.Error())
		os.Exit(1)
	}
	privKey, err := keys.StdPrivKeyGen(derivedPriv, keys.Secp256k1)
	if err != nil {
		logger.Error("failed to generate private key")
		logger.Error(err.Error())
		os.Exit(1)
	}

	// create signer, needs client, privkey, and inflight limit (max limit for txs in mempool)
	signer := NewSigner(broadcastClient, privKey, 500)

	// channels to communicate with signer
	requests := make(chan MsgRequest)

	// signer starts it's own go routines and returns
	responses, err := signer.Run(requests)
	if err != nil {
		logger.Error("failed to start signer")
		logger.Error(err.Error())
		os.Exit(1)
	}

	// log responses, if responses are not read, requests will block
	go func() {
		for {
			// response is not returned until the msg is committed to a block
			response := <-responses

			// error will be set if response is not Code 0 (success) or Code 19 (already in mempool)
			if response.Err != nil {
				fmt.Printf("response code: %d error %s\n", response.Result.Code, response.Err)
				continue
			}

			// code and result are from broadcast, not deliver tx
			// it is up to the caller/requester to check the deliver tx code and deal with failure
			fmt.Printf("response code: %d, hash %s\n", response.Result.Code, response.Result.Hash)
		}
	}()

	// send messages to signer
	for i := 0; i < 1000; i++ {
		fmt.Printf("sending request %d\n", i)
		requests <- MsgRequest{
			Msgs: []sdk.Msg{
				bank.NewMsgSend(GetAccAddress(privKey), config.KavaKeeperAddress, sdk.Coins{sdk.Coin{Denom: "ukava", Amount: sdk.NewInt(1000)}}),
			},
			Fee: authtypes.StdFee{
				Gas: 75000,
			},
			Memo: strconv.Itoa(i),
		}
	}

	select {}
}
