package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/envkey/envkeygo" // XXX could do with comment
	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/kava"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
)

type Config struct {
	BnbRPCURL         string
	KavaRestURL       string
	KavaRPCURL        string
	BnbDeputyAddress  string
	KavaDeputyAddress string
	BnbMnemonics      []string
	KavaMnemonics     []string
}

func loadConfig() Config {
	return Config{
		BnbRPCURL:         os.Getenv("BNB_RPC_URL"),
		KavaRestURL:       os.Getenv("KAVA_REST_URL"),
		KavaRPCURL:        os.Getenv("KAVA_RPC_URL"),
		BnbDeputyAddress:  os.Getenv("BNB_DEPUTY_ADDRESS"),
		KavaDeputyAddress: os.Getenv("KAVA_DEPUTY_ADDRESS"),
		KavaMnemonics:     getSequentialEnvVars("KAVA_MNEMONIC_"),
		BnbMnemonics:      getSequentialEnvVars("BNB_MNEMONIC_"),
	}
}

// XXX could use godoc
func getSequentialEnvVars(prefix string) []string {
	var envVars []string
	for i := 0; ; i++ {
		envVar, found := os.LookupEnv(fmt.Sprintf("%s%d", prefix, i))
		if !found {
			break
		}
		envVars = append(envVars, envVar)
	}
	return envVars
}

func main() {

	cfg := loadConfig()

	// Set global address prefixes
	kavaConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(kavaConfig) // XXX G34 descend only one level of abstraction
	kavaConfig.Seal()

	// XXX G30 functions should do one thing

	// XXX F1 too many arguements
	// XXX G5 duplication
	kavaClaimer := claim.NewKavaClaimer(cfg.KavaRestURL, cfg.KavaRPCURL, cfg.BnbRPCURL, cfg.BnbDeputyAddress, cfg.KavaMnemonics)
	bnbClaimer := claim.NewBnbClaimer(cfg.KavaRestURL, cfg.KavaRPCURL, cfg.BnbRPCURL, cfg.KavaDeputyAddress, cfg.BnbDeputyAddress, cfg.BnbMnemonics)

	ctx := context.Background() // XXX G34 too many levels of abstraction
	kavaClaimer.Run(ctx)
	bnbClaimer.Run(ctx) // XXX G5 duplication

	select {}
}
