package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/alerts/alerter"
	"github.com/kava-labs/go-tools/alerts/auctions"
	"github.com/kava-labs/go-tools/alerts/config"
	"github.com/kava-labs/go-tools/alerts/persistence"
	kava "github.com/kava-labs/kava/app"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

var _serviceName = "AuctionAlerts"

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

		// Load config. If config is not valid, exit with fatal error
		config, err := config.LoadAuctionsConfig(&config.EnvLoader{})
		if err != nil {
			return err
		}

		db, err := persistence.NewDynamoDbPersister(config.DynamoDbTableName, _serviceName, config.KavaRpcUrl)
		if err != nil {
			return err
		}

		// Get last alert to test if we can successfully fetch from DynamoDB
		if _, _, err := db.GetLatestAlert(); err != nil {
			return fmt.Errorf("Failed to fetch alert times from DynamoDB: %v", err)
		}

		// bootstrap kava chain config
		// sets a global cosmos sdk for bech32 prefix
		// required before loading config
		kavaConfig := sdk.GetConfig()
		kava.SetBech32AddressPrefixes(kavaConfig)
		kava.SetBip44CoinType(kavaConfig)
		kavaConfig.Seal()

		// Create slack alerts client
		slackAlerter := alerter.NewSlackAlerter(config.SlackToken)

		logger.With(
			"rpcUrl", config.KavaRpcUrl,
			"UsdThreshold", strings.Split(config.UsdThreshold.String(), ".")[0],
			"Interval", config.Interval.String(),
			"AlertFrequency", config.AlertFrequency.String(),
		).Info("config loaded")

		// Bootstrap rpc http clent
		http, err := rpchttpclient.New(config.KavaRpcUrl, "/websocket")
		if err != nil {
			return err
		}
		http.Logger = logger

		// Create codec for messages
		cdc := kava.MakeCodec()

		// Create rpc client for fetching data
		logger.Info("creating rpc client")
		auctionClient := auctions.NewRpcAuctionClient(http, cdc)

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

			totalValue, err := auctions.CalculateTotalAuctionsUSDValue(data)
			if err != nil {
				logger.Error(err.Error())
				continue
			}

			logger.Info(fmt.Sprintf("Total auction value $%s", totalValue))

			// Total value has not exceeded threshold, continue
			if totalValue.Cmp(config.UsdThreshold.Int) != 1 {
				continue
			}

			warningMsg := fmt.Sprintf(
				"Elevated auction activity:\nTotal collateral value: $%s",
				strings.Split(totalValue.String(), ".")[0],
			)
			logger.Info(warningMsg)

			lastAlert, canAlert, err := alerter.GetAndSaveLastAlert(&db, config.AlertFrequency)
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

			if err := slackAlerter.Warn(
				config.SlackChannelId,
				warningMsg,
			); err != nil {
				logger.Error("Failed to send Slack alert", err.Error())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(auctionsCmd)
	auctionsCmd.AddCommand(runAuctionsCmd)
}
