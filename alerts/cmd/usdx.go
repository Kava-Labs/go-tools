package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/alerts/alerter"
	"github.com/kava-labs/go-tools/alerts/ascendex"
	"github.com/kava-labs/go-tools/alerts/config"
	"github.com/kava-labs/go-tools/alerts/persistence"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	_usdxServiceName          = "UsdxAlerts"
	_ascendexUsdxTickerSymbol = "USDX/USDT"
)

var usdxCmd = &cobra.Command{
	Use:   "usdx",
	Short: "alerter for USDX price on AscendEX",
}

var runUsdxCmd = &cobra.Command{
	Use:     "run",
	Short:   "runs the alerter USDX price on AscendEX",
	Example: "run",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create base logger
		logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

		// Load config. If config is not valid, exit with fatal error
		config, err := config.LoadUsdxConfig(&config.EnvLoader{})
		if err != nil {
			return err
		}

		// "default" for Rpc url, is not used and does not affect USDX alerts
		// but can't be empty
		db, err := persistence.NewDynamoDbPersister(config.DynamoDbTableName, _usdxServiceName, "default")
		if err != nil {
			return err
		}

		// Get last alert to test if we can successfully fetch from DynamoDB
		if _, _, err := db.GetLatestAlert(); err != nil {
			return fmt.Errorf("failed to fetch alert times from DynamoDB: %v", err)
		}

		ascendexClient := ascendex.NewAscendexHttpClient()

		// Create slack alerts client
		slackAlerter := alerter.NewSlackAlerter(config.SlackToken)

		logger.With(
			"UsdxDeviation", config.UsdxDeviation.String(),
			"Interval", config.Interval.String(),
			"AlertFrequency", config.AlertFrequency.String(),
		).Info("config loaded")

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

			summary, err := ascendexClient.Ticker(_ascendexUsdxTickerSymbol)
			if err != nil {
				logger.Error("Failed to fetch AscendEX %v ticker", _ascendexUsdxTickerSymbol, err.Error())
				continue
			}

			logger.Info(fmt.Sprintf("%v last traded price %v",
				_ascendexUsdxTickerSymbol, strings.TrimRight(summary.Close.String(), "0"),
			))

			// ! Price has not exceeded base price +- deviation
			if !exceedsDeviation(summary.Close, config.UsdxBasePrice, config.UsdxDeviation) {
				continue
			}

			lastAlert, found, err := db.GetLatestAlert()
			if err != nil {
				logger.Error("Failed to fetch latest alert time", err.Error())
				continue
			}

			warningMsg := fmt.Sprintf(
				"USDX/USDT price deviation:\nLast traded price T$%v",
				strings.TrimRight(summary.Close.String(), "0"),
			)
			logger.Info(warningMsg)

			// If current time in UTC is before (previous timestamp + alert frequency), skip alert
			if found && time.Now().UTC().Before(lastAlert.Timestamp.Add(config.AlertFrequency)) {
				logger.Info(fmt.Sprintf("Alert already sent within the last %v. (Last was %v, next at %v)",
					config.AlertFrequency,
					lastAlert.Timestamp.Format(time.RFC3339),
					lastAlert.Timestamp.Add(config.AlertFrequency).Format(time.RFC3339),
				))
			} else {
				logger.Info("Sending alert to Slack")

				if err := slackAlerter.Warn(
					config.SlackChannelId,
					warningMsg,
				); err != nil {
					logger.Error("Failed to send Slack alert", err.Error())
				}

				if err := db.SaveAlert(time.Now().UTC()); err != nil {
					logger.Error("Failed to save alert time to DynamoDb", err.Error())
				}
			}
		}
	},
}

func exceedsDeviation(value, base, deviation sdk.Dec) bool {
	return value.Sub(base).Abs().GTE(deviation)
}

func init() {
	rootCmd.AddCommand(usdxCmd)
	usdxCmd.AddCommand(runUsdxCmd)
}
