package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

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

	awsCfg, err := aws_config.LoadDefaultConfig(context.TODO(), aws_config.WithRegion("us-east-1"))
	if err != nil {
		logger.Error("Unable to load AWS SDK config, %v", err)
		os.Exit(1)
	}

	svc := dynamodb.NewFromConfig(awsCfg)

	output, err := svc.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(config.DynamoDbTableName),
		Key: map[string]types.AttributeValue{
			"Id": &types.AttributeValueMemberS{Value: "lastupdate"},
		},
	})

	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	fmt.Println(output)

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

		logger.Info(fmt.Sprintf("Total auction value %s", totalValue))

		// If total value exceeds the set threshold
		// +1 if x > y
		if totalValue.Cmp(config.UsdThreshold.Int) == 1 {
			warningMsg := fmt.Sprintf(
				"Auctions exceeded total USD value!\nTotal: %s USD\nThreshold: %s USD",
				totalValue.String(),
				config.UsdThreshold.String(),
			)

			logger.Info(warningMsg)
			logger.Info("Sending alert to Slack")
			err := slackClient.Warn(
				config.SlackChannelId,
				warningMsg,
			)

			if err != nil {
				logger.Error("Failed to send Slack alert", err.Error())
			}
		}

		// wait for next interval
		time.Sleep(config.Interval)
	}
}
