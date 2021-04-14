package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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

	// client for fetching borrower info
	liquidationClient := NewRpcLiquidationClient(http, cdc)

	//
	// client for broadcasting txs
	//
	broadcastClient := NewRpcBroadcastClient(http, cdc)
	hdPath := keys.CreateHDPath(0, 0)

	derivedPriv, err := keys.StdDeriveKey(config.KavaSignerMnemonic, "", hdPath.String(), keys.Secp256k1)
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
	signer := NewSigner(broadcastClient, privKey, 10)

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

	for {
		// fetch asset and position data using client
		data, err := GetPositionData(liquidationClient)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		// calculate borrowers to liquidate from asset and position data
		borrowersToLiquidate := GetBorrowersToLiquidate(data)
		fmt.Printf("%d borrowers to liquidate\n", len(borrowersToLiquidate))

		// create liquidation msgs
		msgs := CreateLiquidationMsgs(config.KavaKeeperAddress, borrowersToLiquidate)

		// create liquidation transactions
		for _, msg := range msgs {
			fmt.Printf("sending liquidation for %s\n", msg.Borrower.String())

			requests <- MsgRequest{
				Msgs: []sdk.Msg{msg},
				Fee: authtypes.StdFee{
					Amount: sdk.Coins{sdk.Coin{Denom: "ukava", Amount: sdk.NewInt(50000)}},
					Gas:    1000000,
				},
				Memo: "",
			}
		}

		// wait for next interval
		time.Sleep(config.KavaLiquidationInterval)
	}

	select {}
}
