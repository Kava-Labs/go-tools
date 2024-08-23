package cmd

import (
	"fmt"
	"os"
	"sort"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/go-tools/spammer/client"
	"github.com/kava-labs/go-tools/spammer/config"
	"github.com/kava-labs/go-tools/spammer/spammer"
	"github.com/kava-labs/kava/app"
)

var runCmd = &cobra.Command{
	Use:     "run",
	Short:   "runs the spammer",
	Example: "run",
	Run: func(cmd *cobra.Command, args []string) {
		configPath := pflag.String("config", config.DefaultConfigPath, "path to config file")
		pflag.Parse()

		// Load kava claimers
		sdkConfig := sdk.GetConfig()
		app.SetBech32AddressPrefixes(sdkConfig)
		cdc := app.MakeCodec()

		// Load config
		cfg, err := config.GetConfig(cdc, *configPath)
		if err != nil {
			panic(err)
		}

		logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
		kavaClient, err := client.NewKavaClient(cdc, cfg.RPCEndpoint, logger)
		if err != nil {
			panic(err)
		}

		distributorKeyManager, err := keys.NewMnemonicKeyManager(cfg.Mnemonic, app.Bip44CoinType)
		if err != nil {
			panic(err)
		}

		spammerBot, err := spammer.NewSpammer(kavaClient, distributorKeyManager, cfg.NumAccounts)
		if err != nil {
			panic(err)
		}

		// Order messages
		messages := cfg.Messages
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Processor.Order < messages[j].Processor.Order
		})

		// Process messages
		for i, message := range messages {
			log.Infof(fmt.Sprintf("Processing message %d: %s...", i+1, message.Msg.Type()))
			err = spammerBot.ProcessMsg(message)
			if err != nil {
				log.Errorf(err.Error())
			}

			log.Infof(fmt.Sprintf("Waiting %d seconds...", message.Processor.AfterWaitSeconds))
			time.Sleep(time.Duration(message.Processor.AfterWaitSeconds) * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
