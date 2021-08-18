package main

import (
	"fmt"
	"os"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	kava "github.com/kava-labs/kava/app"
	"github.com/tendermint/tendermint/libs/log"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

const (
	kavaRpcUrlEnvKey = "KAVA_RPC_URL"
	slackToken       = "SLACK_TOKEN"
	slackChannelId   = "SLACK_CHANNEL_ID"
	interval         = "INTERVAL"
	usdThreshold     = "USD_THRESHOLD"
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
		"Interval", config.Interval.String(),
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

	for {
		data, err := GetAuctionData(auctionClient)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		logger.Info(fmt.Sprintf("checking %d auctions", len(data.Auctions)))

		fmt.Println(data.Assets)

		totalValue := sdk.NewDec(0)

		for _, auction := range data.Auctions {
			lot := auction.GetLot()
			assetInfo, ok := data.Assets[lot.Denom]
			if !ok {
				logger.Error("Missing asset info for %s", lot.Denom)
				os.Exit(1)
			}

			usdValue := calculateUSDValue(lot, assetInfo)

			totalValue = totalValue.Add(usdValue)

			fmt.Println(lot.String())
		}

		fmt.Println("Total auction value", totalValue)

		// wait for next interval
		time.Sleep(config.Interval)
	}
}
