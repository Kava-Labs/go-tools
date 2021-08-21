package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"

	slack_alerts "github.com/kava-labs/go-tools/slack-alerts"
	kava "github.com/kava-labs/kava/app"
	"github.com/tendermint/tendermint/libs/log"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

const (
	kavaRpcUrlEnvKey        = "KAVA_RPC_URL"
	slackTokenEnvKey        = "SLACK_TOKEN"
	slackChannelIdEnvKey    = "SLACK_CHANNEL_ID"
	intervalEnvKey          = "INTERVAL"
	alertFrequencyEnvKey    = "ALERT_FREQUENCY"
	usdThresholdEnvKey      = "USD_THRESHOLD"
	dynamoDbTableNameEnvKey = "DYNAMODB_TABLE_NAME"
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

	db, err := NewDb()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

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
	slackClient := slack_alerts.NewSlackAlerter(config.SlackToken)

	logger.With(
		"rpcUrl", config.KavaRpcUrl,
		"Interval", config.Interval.String(),
		"AlertFrequency", config.AlertFrequency.String(),
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

			fmt.Println(lot, assetInfo, usdValue)
		}

		logger.Info(fmt.Sprintf("Total auction value $%s", totalValue))

		// If total value exceeds the set threshold
		// +1 if x > y
		if totalValue.Cmp(config.UsdThreshold.Int) == 1 {
			lastAlert, found, err := db.GetLatestAlert(config.DynamoDbTableName, config.KavaRpcUrl)
			if err != nil {
				logger.Error("Failed to fetch latest alert time", err.Error())
				continue
			}

			fmt.Println(lastAlert)

			warningMsg := fmt.Sprintf(
				"Elevated auction activity:\nTotal collateral value: $%s",
				strings.Split(totalValue.String(), ".")[0],
			)
			logger.Info(warningMsg)

			// If current time in UTC is before (previous timestamp + alert frequency), skip alert
			if found && time.Now().UTC().Before(lastAlert.Timestamp.Add(config.AlertFrequency)) {
				logger.Info(fmt.Sprintf("Alert already sent within the last %v. (Last was %v)",
					config.AlertFrequency,
					lastAlert.Timestamp.Format(time.RFC3339),
				))
			} else {
				logger.Info("Sending alert to Slack")

				if err := slackClient.Warn(
					config.SlackChannelId,
					warningMsg,
				); err != nil {
					logger.Error("Failed to send Slack alert", err.Error())
				}

				if _, err := db.SaveAlert(config.DynamoDbTableName, config.KavaRpcUrl, time.Now().UTC()); err != nil {
					logger.Error("Failed to save alert time to DynamoDb", err.Error())
				}
			}
		}

		// wait for next interval
		time.Sleep(config.Interval)
	}
}
