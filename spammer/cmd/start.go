package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	// "github.com/spf13/viper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bip39 "github.com/cosmos/go-bip39"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/go-tools/spammer/client"
	"github.com/kava-labs/go-tools/spammer/spammer"
	"github.com/kava-labs/kava/app"
)

// const flagInFile = "file"

const (
	mnemonic    = ""
	rpcAddr     = "http://3.236.68.204:26657"
	numAccounts = 1000
)

var (
	amountPerAddress = sdk.NewCoins(sdk.NewInt64Coin("xrpb", 200000000000), sdk.NewInt64Coin("ukava", 10000000), sdk.NewInt64Coin("hard", 50000000), sdk.NewInt64Coin("bnb", 200000000))
	cdpCollateral    = sdk.NewInt64Coin("xrpb", 100000000000)
	collateralType   = "xrpb-a"
	cdpPrincipal     = sdk.NewInt64Coin("usdx", 260000000)
	hardDeposit      = sdk.NewCoins(sdk.NewInt64Coin("usdx", 260000000), sdk.NewInt64Coin("xrpb", 100000000000), sdk.NewInt64Coin("bnb", 200000000))
	hardBorrow       = sdk.NewCoins(sdk.NewInt64Coin("bnb", 100000000))
)

// startCmd : represents the start command
var startCmd = &cobra.Command{
	Use:     "run",
	Short:   "runs the spammer",
	Example: "run",
	Run: func(cmd *cobra.Command, args []string) {
		// Parse flags
		// filePath := viper.GetString(flagInFile)
		// if strings.TrimSpace(filePath) == "" {
		// 	log.Fatal("invalid --file flag")
		// }

		// Start Kava HTTP client
		config := sdk.GetConfig()
		app.SetBech32AddressPrefixes(config)
		cdc := app.MakeCodec()

		logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
		kavaClient, err := client.NewKavaClient(cdc, rpcAddr, logger)
		if err != nil {
			panic(err)
		}

		// Set up accounts
		distributorKeyManager, err := keys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
		if err != nil {
			panic(err)
		}

		// Set up accounts
		accounts, err := genNewAccounts(numAccounts)
		if err != nil {
			fmt.Println(err)
		}

		spamBot := spammer.NewSpammer(kavaClient, distributorKeyManager, accounts)

		// Distribute coins to spammer's accounts
		err = spamBot.DistributeCoins(amountPerAddress)
		if err != nil {
			fmt.Println(err)
		}

		// Wait for the distribution tx to be confirmed
		time.Sleep(20 * time.Second)

		// Each account sends a CDP creation tx
		err = spamBot.OpenCDPs(cdpCollateral, cdpPrincipal, collateralType)
		if err != nil {
			fmt.Println(err)
		}

		// Wait for the create CDP txs to be confirmed
		time.Sleep(120 * time.Second)

		// Each account sends a Hard deposit tx
		err = spamBot.HardDeposits(hardDeposit)
		if err != nil {
			fmt.Println(err)
		}

		// Wait for the Deposit txs to be confirmed
		time.Sleep(45 * time.Second)

		// Each account sends a Hard borrow tx
		err = spamBot.HardBorrows(hardBorrow)
		if err != nil {
			fmt.Println(err)
		}
	},
}

// init : prepares required flags and adds them to the start cmd
func init() {
	rootCmd.AddCommand(startCmd)

	// // Add flags and mark as required
	// startCmd.PersistentFlags().String(flagInFile, "", "Path to start file")
	// startCmd.MarkFlagRequired(flagInFile)

	// // Bind flags
	// viper.BindPFlag(flagInFile, startCmd.Flag(flagInFile))
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
