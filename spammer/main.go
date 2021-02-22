package main

import (
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bip39 "github.com/cosmos/go-bip39"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/app"

	"github.com/kava-labs/go-tools/spammer/client"
	"github.com/kava-labs/go-tools/spammer/spammer"
)

const (
	mnemonic = "fragile flip puzzle adjust mushroom gas minimum maid love coach brush cattle match analyst oak spell blur thunder unfair inch mother park toilet toddler"
	rpcAddr  = "tcp://localhost:26657"
)

func main() {
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
	accounts, err := genNewAccounts(2)
	if err != nil {
		fmt.Println(err)
	}

	spamBot := spammer.NewSpammer(kavaClient, distributorKeyManager, accounts)

	// Distribute coins to spammer's accounts
	err = spamBot.DistributeCoins(100000000) // 100 KAVA per address
	if err != nil {
		fmt.Println(err)
	}

	// Each account sends a CDP creation tx
	err = spamBot.OpenCDPs()
	if err != nil {
		fmt.Println(err)
	}

	// Each account sends a Hard deposit tx
	err = spamBot.HardDeposits()
	if err != nil {
		fmt.Println(err)
	}

	// Each account sends a Hard borrow tx
	err = spamBot.HardBorrows()
	if err != nil {
		fmt.Println(err)
	}
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
