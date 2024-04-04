package cmd

import (
	"fmt"
	"os"

	"github.com/kava-labs/go-tools/alerts/alerter"
	"github.com/kava-labs/go-tools/alerts/config"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/log"
)

var (
	rootCmd = &cobra.Command{
		Use:   "alerts",
		Short: "alerter for the Kava blockchain",
	}
	testCmd = &cobra.Command{
		Use:   "slack-test",
		Short: "test sending of message to slack",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := config.LoadBaseConfig(&config.EnvLoader{}, log.NewTMLogger(log.NewSyncWriter(os.Stdout)))
			if err != nil {
				return err
			}
			slackAlerter := alerter.NewSlackAlerter(config.SlackWebhookUrl)
			return slackAlerter.Warn("testing the alerts repo slack integration")
		},
	}
)

func init() {
	rootCmd.AddCommand(testCmd)
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
