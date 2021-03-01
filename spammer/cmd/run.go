package cmd

import (
	"fmt"
	"os"
	"sort"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bip39 "github.com/cosmos/go-bip39"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/go-tools/spammer/client"
	"github.com/kava-labs/go-tools/spammer/config"
	"github.com/kava-labs/go-tools/spammer/spammer"
	"github.com/kava-labs/kava/app"
)

// TODO: use these values in the config file
var (
	amountPerAddress = sdk.NewCoins(sdk.NewInt64Coin("xrpb", 200000000000), sdk.NewInt64Coin("ukava", 10000000), sdk.NewInt64Coin("hard", 50000000), sdk.NewInt64Coin("bnb", 200000000))
	cdpCollateral    = sdk.NewInt64Coin("xrpb", 100000000000)
	collateralType   = "xrpb-a"
	cdpPrincipal     = sdk.NewInt64Coin("usdx", 260000000)
	hardDeposit      = sdk.NewCoins(sdk.NewInt64Coin("usdx", 260000000), sdk.NewInt64Coin("xrpb", 100000000000), sdk.NewInt64Coin("bnb", 200000000))
	hardBorrow       = sdk.NewCoins(sdk.NewInt64Coin("bnb", 100000000))
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

		// Set up accounts
		distributorKeyManager, err := keys.NewMnemonicKeyManager(cfg.Mnemonic, app.Bip44CoinType)
		if err != nil {
			panic(err)
		}

		// Set up accounts
		accounts, err := genNewAccounts(cfg.NumAccounts)
		if err != nil {
			log.Errorf(err.Error())
		}

		spamBot := spammer.NewSpammer(kavaClient, distributorKeyManager, accounts)

		// Order messages
		messages := cfg.Messages
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Processor.Order < messages[j].Processor.Order
		})

		for i, message := range messages {
			log.Infof(fmt.Sprintf("Processing message %d: %s...", i+1, message.Msg.Type()))
			err = spamBot.ProcessMsg(message)
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

func genNewAccounts(count int) ([]keys.KeyManager, error) {
	var kavaKeys []keys.KeyManager
	for i := 0; i < count; i++ {
		entropySeed, err := bip39.NewEntropy(256)
		if err != nil {
			return kavaKeys, err
		}

		mnemonic, err := bip39.NewMnemonic(entropySeed)
		if err != nil {
			return kavaKeys, err
		}

		keyManager, err := keys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
		if err != nil {
			return kavaKeys, err
		}
		kavaKeys = append(kavaKeys, keyManager)
	}

	return kavaKeys, nil
}
