package main

import (
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	client := NewRpcAuctionClient(http, cdc)

	logger.Info("getting auction data")
	data, err := GetAuctionData(client)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	auctionBids := GetBids(data, config.KavaKeeperAddress, config.ProfitMargin)

	msgs := CreateBidMsgs(config.KavaKeeperAddress, auctionBids)

	// print number of borrowers that need to be liquidated
	fmt.Printf("%d bids to make\n", len(auctionBids))
	fmt.Println(msgs)
}
