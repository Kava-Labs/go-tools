package claimer

import (
	"context"
	"os"
	"strings"
	"time"

	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bkeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/keys"
	kava "github.com/kava-labs/kava/app"
	tmlog "github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

var JobQueueSize = 1000

// KavaClaimer is a worker that sends claim transactions on Kava
type KavaClaimer struct {
	Keybase keys.KeyManager
	Status  bool
}

type Dispatcher struct {
	jobQueue <-chan server.ClaimJob
}

func NewDispatcher() Dispatcher {
	jobQueue := make(chan server.ClaimJob, JobQueueSize)
	return Dispatcher{
		jobQueue: jobQueue,
	}
}

func (d Dispatcher) Start(ctx context.Context, c config.Config) {
	// Load kava claimers
	sdkConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(sdkConfig)
	cdc := kava.MakeCodec()

	// SETUP CLAIMERS --------------------------
	var kavaClaimers []KavaClaimer
	for _, kavaMnemonic := range c.Kava.Mnemonics {
		kavaClaimer := KavaClaimer{}
		keyManager, err := keys.NewMnemonicKeyManager(kavaMnemonic, kava.Bip44CoinType)
		if err != nil {
			log.Error(err)
		}
		kavaClaimer.Keybase = keyManager
		kavaClaimer.Status = true
		kavaClaimers = append(kavaClaimers, kavaClaimer)
	}

	// SETUP KAVA CLIENT --------------------------
	// Start Kava HTTP client
	http, err := rpcclient.New(c.Kava.Endpoint, "/websocket")
	if err != nil {
		panic(err)
	}
	http.Logger = tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	err = http.Start()
	if err != nil {
		panic(err)
	}

	// SETUP BNB CLIENT --------------------------
	// Set up Binance Chain client
	bncNetwork := btypes.TestNetwork
	if c.BinanceChain.ChainID == "Binance-Chain-Tigris" {
		bncNetwork = btypes.ProdNetwork
	}
	bnbClient := brpc.NewRPCClient(c.BinanceChain.Endpoint, bncNetwork)
	bnbKeyManager, err := bkeys.NewMnemonicKeyManager(c.BinanceChain.Mnemonic)
	if err != nil {
		panic(err)
	}
	bnbClient.SetKeyManager(bnbKeyManager)

	sem := semaphore.NewWeighted(int64(len(kavaClaimers)))

	// RUN WORKERS --------------------------
	for {
		select {
		case <-ctx.Done():
			return
		case claim := <-d.jobQueue:
			switch strings.ToUpper(claim.TargetChain) {
			case server.TargetKava:
				if err := sem.Acquire(ctx, 1); err != nil {
					log.Error(err)
					return
				}

				go func() {
					defer sem.Release(1)
					Retry(10, 20*time.Second, func() (err ClaimError) {
						err = claimOnKava(c.Kava, http, claim, cdc, kavaClaimers)
						return
					})
				}()
				break
			case server.TargetBinance, server.TargetBinanceChain:
				go func() {
					Retry(10, 15*time.Second, func() (err ClaimError) {
						err = claimOnBinanceChain(bnbClient, claim)
						return
					})
				}()
				break
			}
		}
	}
}

func (d Dispatcher) JobQueue() chan server.ClaimJob { return d.jobQueue }
