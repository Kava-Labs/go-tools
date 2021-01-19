package claimer

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bkeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/go-sdk/keys"
	kava "github.com/kava-labs/kava/app"
	log "github.com/sirupsen/logrus"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"golang.org/x/sync/semaphore"

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
	config   config.Config
	jobQueue chan server.ClaimJob
	cdc      *codec.Codec
}

func NewDispatcher(cfg config.Config) Dispatcher {
	jobQueue := make(chan server.ClaimJob, JobQueueSize)
	cdc := kava.MakeCodec()
	return Dispatcher{
		config:   cfg,
		jobQueue: jobQueue,
		cdc:      cdc,
	}
}

func (d Dispatcher) Start(ctx context.Context) {
	// SETUP CLAIMERS --------------------------
	var kavaClaimers []KavaClaimer
	for _, kavaMnemonic := range d.config.Kava.Mnemonics {
		kavaClaimer := KavaClaimer{}
		keyManager, err := keys.NewMnemonicKeyManager(kavaMnemonic, kava.Bip44CoinType)
		if err != nil {
			log.Error(err)
		}
		kavaClaimer.Keybase = keyManager
		kavaClaimer.Status = true
		kavaClaimers = append(kavaClaimers, kavaClaimer)
	}

	// Start Kava HTTP client
	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	kavaClient, err := NewKavaClient(d.cdc, d.config.Kava.Endpoint, logger)
	if err != nil {
		panic(err)
	}

	// SETUP BNB CLIENT --------------------------
	// Set up Binance Chain client
	bncNetwork := btypes.TestNetwork
	if d.config.BinanceChain.ChainID == "Binance-Chain-Tigris" {
		bncNetwork = btypes.ProdNetwork
	}
	bnbClient := brpc.NewRPCClient(d.config.BinanceChain.Endpoint, bncNetwork)
	bnbKeyManager, err := bkeys.NewMnemonicKeyManager(d.config.BinanceChain.Mnemonic)
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
					Retry(10, 20*time.Second, func() error {
						return claimOnKava(d.config.Kava, kavaClient, claim, kavaClaimers)
					})
				}()
			case server.TargetBinance, server.TargetBinanceChain:
				go func() {
					Retry(10, 15*time.Second, func() error {
						return claimOnBinanceChain(bnbClient, claim)
					})
				}()
			}
		}
	}
}

func (d Dispatcher) JobQueue() chan server.ClaimJob { return d.jobQueue }
