package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/alerts/alerter"
	"github.com/kava-labs/go-tools/alerts/auctions"
	"github.com/kava-labs/go-tools/alerts/config"
	"github.com/kava-labs/go-tools/alerts/persistence"
	kava "github.com/kava-labs/kava/app"
	kavagrpc "github.com/kava-labs/kava/client/grpc"
	"github.com/spf13/cobra"
)

var _auctionsServiceName = "AuctionAlerts"
var _inefficientAuctionServiceName = "InefficientAuctionAlerts"

var auctionsCmd = &cobra.Command{
	Use:   "auctions",
	Short: "alerter for auctions on the Kava blockchain",
}

var runAuctionsCmd = &cobra.Command{
	Use:     "run",
	Short:   "runs the alerter for auctions on the Kava blockchain",
	Example: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create base logger
		logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

		// Load config. If config is not valid, exit with error
		config, err := config.LoadAuctionsConfig(&config.EnvLoader{}, logger)
		if err != nil {
			return err
		}

		// Create a new alert persisted backed with DynamoDB. If AWS config is
		// invalid, exits with error
		auctionDB, err := persistence.NewDynamoDbPersister(config.DynamoDbTableName, _auctionsServiceName, config.KavaGrpcUrl)
		if err != nil {
			return err
		}

		// Get last alert to test if we can successfully fetch from DynamoDB
		if _, _, err := auctionDB.GetLatestAlert(); err != nil {
			return fmt.Errorf("failed to fetch auction alert times from DynamoDB: %v", err)
		}

		// bootstrap kava chain config
		// sets a global cosmos sdk for bech32 prefix
		// required before loading config
		kavaConfig := sdk.GetConfig()
		kava.SetBech32AddressPrefixes(kavaConfig)
		kava.SetBip44CoinType(kavaConfig)
		kavaConfig.Seal()

		// Create slack alerts client
		slackAlerter := alerter.NewSlackAlerter(config.SlackWebhookUrl)

		logger.With(
			"grpcUrl", config.KavaGrpcUrl,
			"UsdThreshold", strings.Split(config.UsdThreshold.String(), ".")[0],
			"Interval", config.Interval.String(),
			"AlertFrequency", config.AlertFrequency.String(),
		).Info("config loaded")

		// Create codec for messages
		encodingConfig := kava.MakeEncodingConfig()

		// Bootstrap grpc client
		grpcClient, err := kavagrpc.NewClient(config.KavaGrpcUrl)
		if err != nil {
			return fmt.Errorf("failed to create grpc client: %v", err)
		}

		// Create rpc client for fetching data
		logger.Info("creating rpc client")
		auctionClient := auctions.NewGrpcAuctionClient(grpcClient, encodingConfig.Marshaler)

		firstIteration := true

		for {
			// Wait for next interval after the first loop. This is done at the
			// beginning so that any following `continue` statements will not
			// continue the loop immediately.
			if !firstIteration {
				time.Sleep(config.Interval)
			} else {
				firstIteration = false
			}

			data, err := auctions.GetAuctionData(auctionClient)
			if err != nil {
				logger.Error(err.Error())
				continue
			}

			logger.Info(fmt.Sprintf("checking %d auctions", len(data.Auctions)))

			inEfficientAuctions, err := auctions.CheckInefficientAuctions(data, config.InefficientAuctionUSDThreshold, config.InefficientRatio, config.InefficientTimeRemaining)
			if err != nil {
				logger.Error(err.Error())
				continue
			}

			for _, auction := range inEfficientAuctions {
				inefficientAuctionDB, err := persistence.NewDynamoDbPersister(config.DynamoDbTableName, _inefficientAuctionServiceName+fmt.Sprint(auction.GetID()), config.KavaGrpcUrl)
				if err != nil {
					return err
				}
				usdBid := auctions.CalculateUSDValue(auction.GetBid(), data.Assets[auction.GetBid().Denom])
				usdLot := auctions.CalculateUSDValue(auction.GetLot(), data.Assets[auction.GetLot().Denom])
				if _, _, err := inefficientAuctionDB.GetLatestAlert(); err != nil {
					return fmt.Errorf("failed to fetch inefficient auction alert time from DynamoDB: %v", err)
				}
				warningMsg := fmt.Sprintf(
					"Inefficient auction:\nAuction ID: %d\nBid: %s (USD Value: $%s)\nLot: %s (USD Value: $%s)\nEnd time: %s\nTime Remaining: %s\n",
					auction.GetID(), auction.GetBid(), usdBid.TruncateInt(), auction.GetLot(), usdLot.TruncateInt(), auction.GetEndTime().Round(time.Minute), time.Until(auction.GetEndTime()).Round(time.Minute),
				)
				logger.Info(warningMsg)

				lastAlert, canAlert, err := alerter.GetAndSaveLastAlert(&inefficientAuctionDB, config.AlertFrequency)
				if err != nil {
					logger.Error("Failed to check alert interval: %v", err.Error())
					continue
				}

				// If current time in UTC is before (previous timestamp + alert frequency), skip alert
				if !canAlert {
					logger.Info(fmt.Sprintf("Alert already sent within the last %v. (Last was %v, next at %v)",
						config.AlertFrequency,
						lastAlert.Timestamp.Format(time.RFC3339),
						lastAlert.Timestamp.Add(config.AlertFrequency).Format(time.RFC3339),
					))

					continue
				}

				logger.Info("Sending alert to Slack")

				if err := slackAlerter.Warn(warningMsg); err != nil {
					logger.Error("Failed to send Slack alert", err.Error())
				}

			}

			totalValue, err := auctions.CalculateTotalAuctionsUSDValue(data)
			if err != nil {
				logger.Error(err.Error())
				continue
			}

			logger.Info(fmt.Sprintf("Total auction value $%s", totalValue))

			// Total value has not exceeded threshold, continue
			if totalValue.LT(config.UsdThreshold) {
				continue
			}

			warningMsg := fmt.Sprintf(
				"Elevated auction activity:\nTotal collateral value: $%s",
				strings.Split(totalValue.String(), ".")[0],
			)
			logger.Info(warningMsg)

			lastAlert, canAlert, err := alerter.GetAndSaveLastAlert(&auctionDB, config.AlertFrequency)
			if err != nil {
				logger.Error("Failed to check alert interval: %v", err.Error())
				continue
			}

			// If current time in UTC is before (previous timestamp + alert frequency), skip alert
			if !canAlert {
				logger.Info(fmt.Sprintf("Alert already sent within the last %v. (Last was %v, next at %v)",
					config.AlertFrequency,
					lastAlert.Timestamp.Format(time.RFC3339),
					lastAlert.Timestamp.Add(config.AlertFrequency).Format(time.RFC3339),
				))

				continue
			}

			logger.Info("Sending alert to Slack")

			if err := slackAlerter.Warn(warningMsg); err != nil {
				logger.Error("Failed to send Slack alert", err.Error())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(auctionsCmd)
	auctionsCmd.AddCommand(runAuctionsCmd)
}
