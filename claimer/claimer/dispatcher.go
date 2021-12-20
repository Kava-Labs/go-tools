package claimer

import (
	"context"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/cosmos/cosmos-sdk/codec"
	tmlog "github.com/tendermint/tendermint/libs/log"

	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bkeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
	kava "github.com/kava-labs/kava/app"
)

var JobQueueSize = 1000

type Dispatcher struct {
	config   config.Config
	jobQueue chan server.ClaimJob
	cdc      *codec.LegacyAmino
}

func NewDispatcher(cfg config.Config) Dispatcher {
	jobQueue := make(chan server.ClaimJob, JobQueueSize)

	tApp := kava.NewTestApp()
	cdc := tApp.LegacyAmino()

	return Dispatcher{
		config:   cfg,
		jobQueue: jobQueue,
		cdc:      cdc,
	}
}

func (d Dispatcher) Start(ctx context.Context) {
	// Setup Mnemonics
	kavaKeys := make(chan keys.KeyManager, len(d.config.Kava.Mnemonics))
	for _, kavaMnemonic := range d.config.Kava.Mnemonics {
		keyManager, err := keys.NewMnemonicKeyManager(kavaMnemonic, kava.Bip44CoinType)
		if err != nil {
			log.Error(err)
		}
		kavaKeys <- keyManager
	}

	// Start Kava HTTP client
	logger := tmlog.NewTMLogger(tmlog.NewSyncWriter(os.Stdout))
	kavaClient, err := NewKavaClient(d.cdc, d.config.Kava.Endpoint, logger)
	if err != nil {
		panic(err)
	}

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

	// Run Workers
	for {
		select {
		case <-ctx.Done():
			return
		case claim := <-d.jobQueue:
			switch strings.ToUpper(claim.TargetChain) {
			case server.TargetKava:
				// fetch an available mnemonic, waiting if none available // TODO should respect ctx
				key := <-kavaKeys

				go func() {
					// release the mnemonic when done
					defer func() { kavaKeys <- key }()
					Retry(30, 20*time.Second, func() error {
						return claimOnKava(d.config.Kava, kavaClient, claim, key)
					})
				}()
			case server.TargetBinance, server.TargetBinanceChain:
				// TODO make binance safe for concurrent requests
				go func() {
					Retry(30, 20*time.Second, func() error {
						return claimOnBinanceChain(bnbClient, claim)
					})
				}()
			}
		}
	}
}

func (d Dispatcher) JobQueue() chan server.ClaimJob { return d.jobQueue }
