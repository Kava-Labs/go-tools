package main

import (
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"

	// slack_alerts "github.com/kava-labs/go-tools/slack-alerts"
	kava "github.com/kava-labs/kava/app"
	"github.com/tendermint/tendermint/libs/log"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

const (
	kavaRpcUrlEnvKey             = "KAVA_RPC_URL"
	slackTokenEnvKey             = "SLACK_TOKEN"
	slackChannelIdEnvKey         = "SLACK_CHANNEL_ID"
	intervalEnvKey               = "INTERVAL"
	alertFrequencyEnvKey         = "ALERT_FREQUENCY"
	spreadPercentThresholdEnvKey = "SPREAD_PERCENT_THRESHOLD"
	dynamoDbTableNameEnvKey      = "DYNAMODB_TABLE_NAME"
)

func main() {
	// create base logger
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

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

	/*
		db, err := NewDb()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	*/

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

	// Create slack alerts client
	// slackClient := slack_alerts.NewSlackAlerter(config.SlackToken)

	logger.With(
		"rpcUrl", config.KavaRpcUrl,
		"Interval", config.Interval.String(),
		"AlertFrequency", config.AlertFrequency.String(),
		"SpreadThresholdPercent", config.SpreadPercentThreshold,
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
	swapClient := NewRpcSwapClient(http, cdc)

	data, err := GetSwapPoolData(swapClient)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	fmt.Println(data)
}
