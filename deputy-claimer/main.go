package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
)

type Config struct {
	BnbRPCURL        string
	KavaRestURL      string
	BnbDeputyAddress string
	KavaMnemonics    []string
}

func main() {

	cfg := Config{
		BnbRPCURL:        os.Getenv("BNB_RPC_URL"),
		KavaRestURL:      os.Getenv("BNB_REST_URL"),
		BnbDeputyAddress: os.Getenv("BNB_DEPUTY_ADDRESS"),
	}
	for i := 0; ; i++ {
		mnemonic, found := os.LookupEnv(fmt.Sprintf("KAVA_MNEMONIC_%d", i))
		if !found {
			break
		}
		cfg.KavaMnemonics = append(cfg.KavaMnemonics, mnemonic)
	}

	for {
		err := claim.RunKava(cfg.KavaRestURL, cfg.BnbRPCURL, cfg.BnbDeputyAddress, cfg.KavaMnemonics)
		if err != nil {
			log.Println(err)
		}
		time.Sleep(5 * time.Minute)
	}
	// repeat for bnb
}
