package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	kava "github.com/kava-labs/kava/app"
	"github.com/tendermint/tendermint/libs/log"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

const (
	kavaRpcUrlEnvKey = "KAVA_RPC_URL"
	mnemonicEnvKey   = "KEEPER_MNEMONIC"
	profitMargin     = "BID_MARGIN"
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
		"bidInterval", config.KavaBidInterval.String(),
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
	// create rpc client for fetching data
	// required for bidding
	//
	logger.Info("creating rpc client")
	auctionClient := NewRpcAuctionClient(http, cdc)

	//
	// client for broadcasting txs
	//
	broadcastClient := NewRpcBroadcastClient(http, cdc)
	params := *hd.NewFundraiserParams(0, 459, 0)
	hdPath := params.String()

	derivedPriv, err := keys.StdDeriveKey(config.KavaKeeperMnemonic, "", hdPath, keys.Secp256k1)
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
	logger.Info(fmt.Sprintf("signing address: %s", sdk.AccAddress(privKey.PubKey().Address()).String()))

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
		data, err := GetAuctionData(auctionClient)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		info, err := auctionClient.GetInfo()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
		logger.Info(fmt.Sprintf("latest height: %d", info.LatestHeight))

		logger.Info(fmt.Sprintf("checking %d auctions", len(data.Auctions)))

		auctionBids := GetBids(data, sdk.AccAddress(privKey.PubKey().Address()), config.ProfitMargin)

		msgs := CreateBidMsgs(sdk.AccAddress(privKey.PubKey().Address()), auctionBids)

		logger.Info(fmt.Sprintf("creating %d bids", len(msgs)))

		for _, msg := range msgs {
			requests <- MsgRequest{
				Msgs: []sdk.Msg{msg},
				Fee: authtypes.StdFee{
					Amount: sdk.Coins{sdk.Coin{Denom: "ukava", Amount: sdk.NewInt(10000)}},
					Gas:    200000,
				},
				Memo: "",
			}
		}

		// wait for next interval
		time.Sleep(config.KavaBidInterval)
	}
}
