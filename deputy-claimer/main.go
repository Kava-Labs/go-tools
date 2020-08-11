package main

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/kava"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
)

type Config struct {
	BnbRPCURL        string
	KavaRestURL      string
	KavaRPCURL       string
	BnbDeputyAddress string
	KavaMnemonics    []string
}

func main() {

	cfg := Config{
		BnbRPCURL:        os.Getenv("BNB_RPC_URL"),
		KavaRestURL:      os.Getenv("KAVA_REST_URL"),
		KavaRPCURL:       os.Getenv("KAVA_RPC_URL"),
		BnbDeputyAddress: os.Getenv("BNB_DEPUTY_ADDRESS"),
	}
	for i := 0; ; i++ {
		mnemonic, found := os.LookupEnv(fmt.Sprintf("KAVA_MNEMONIC_%d", i))
		if !found {
			break
		}
		cfg.KavaMnemonics = append(cfg.KavaMnemonics, mnemonic)
	}

	// Set global address prefixes
	kavaConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(kavaConfig)
	kavaConfig.Seal()

	kavaClaimer := claim.NewKavaClaimer(cfg.KavaRestURL, cfg.KavaRPCURL, cfg.BnbRPCURL, cfg.BnbDeputyAddress, cfg.KavaMnemonics)
	ctx := context.Background()
	kavaClaimer.Run(ctx)

	// repeat for bnb

	select {}
}
