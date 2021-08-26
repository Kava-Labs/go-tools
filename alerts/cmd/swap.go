package cmd

import (
	"fmt"
	"os"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/alerts/alerter"
	"github.com/kava-labs/go-tools/alerts/config"
	"github.com/kava-labs/go-tools/alerts/persistence"
	"github.com/kava-labs/go-tools/alerts/swap"
	kava "github.com/kava-labs/kava/app"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

var _swapArbitrageServiceName = "SwapArbitrageAlerts"

var swapCmd = &cobra.Command{
	Use:   "swap",
	Short: "alerter for swap on the Kava blockchain",
}

var swapArbitrageCmd = &cobra.Command{
	Use:   "arbitrage",
	Short: "alerter for swap arbitrage on the Kava blockchain",
}

var runArbitrageSwapCmd = &cobra.Command{
	Use:     "run",
	Short:   "runs the alerter for swap arbitrage on the Kava blockchain",
	Example: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create base logger
		logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

		// Load config. If config is not valid, exit with fatal error
		config, err := config.LoadArbitrageConfig(&config.EnvLoader{})
		if err != nil {
			return err
		}

		db, err := persistence.NewDynamoDbPersister(config.DynamoDbTableName, _swapArbitrageServiceName, config.KavaRpcUrl)
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
			"SpreadPercentThreshold", config.SpreadPercentThreshold,
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

		swapClient := swap.NewRpcSwapClient(http, cdc)

		frequencyLimiter := alerter.NewFrequencyLimiter(&db, config.AlertFrequency)

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

			// data, err := auctions.GetAuctionData(auctionClient)
			pools, err := swap.GetPoolsData(swapClient)
			if err != nil {
				logger.Error(err.Error())
				continue
			}

			logger.Info(fmt.Sprintf("Pools: %v", pools))
			return nil

			logger.Info(fmt.Sprintf("checking %d pools", len(pools)))

			// Spreads have not exceeded threshold, continue
			// if totalValue.Cmp(config.UsdThreshold.Int) != 1 {
			// 	continue
			// }

			msg := fmt.Sprintf("Swap spread diverged")
			logger.Info(msg)

			frequencyLimiter.Exec(func() error {
				// If last alert has past alert frequency
				logger.Info("Sending alert to Slack")

				if err := slackAlerter.Info(
					config.SlackChannelId,
					msg,
				); err != nil {
					logger.Error("Failed to send Slack alert", err.Error())
				}

				return nil
			}, func(lastAlert persistence.AlertTime) error {
				// Last alert is within alert frequency, only log locally
				logger.Info(fmt.Sprintf("Alert already sent within the last %v. (Last was %v, next at %v)",
					config.AlertFrequency,
					lastAlert.Timestamp.Format(time.RFC3339),
					lastAlert.Timestamp.Add(config.AlertFrequency).Format(time.RFC3339),
				))

				return nil
			})
		}
	},
}

func init() {
	rootCmd.AddCommand(swapCmd)
	swapCmd.AddCommand(swapArbitrageCmd)
	swapArbitrageCmd.AddCommand(runArbitrageSwapCmd)
}
