package claimer

import (
	"context"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	brpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	btypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bkeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	log "github.com/sirupsen/logrus"

	kava "github.com/kava-labs/kava/app"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/types"
)

var JobQueueSize = 1000

type Dispatcher struct {
	config     config.Config
	jobQueue   chan types.ClaimJob
	KavaClient grpcKavaClient
	BnbClient  *brpc.HTTP
}

func NewDispatcher(cfg config.Config) *Dispatcher {
	jobQueue := make(chan types.ClaimJob, JobQueueSize)

	return &Dispatcher{
		config:   cfg,
		jobQueue: jobQueue,
	}
}

func (d *Dispatcher) Start(ctx context.Context) {
	// Setup Mnemonics
	kavaKeys := make(chan cryptotypes.PrivKey, len(d.config.Kava.Mnemonics))
	for _, kavaMnemonic := range d.config.Kava.Mnemonics {
		hdPath := hd.CreateHDPath(kava.Bip44CoinType, 0, 0)
		privKeyBytes, err := hd.Secp256k1.Derive()(kavaMnemonic, "", hdPath.String())
		if err != nil {
			panic(err)
		}
		kavaKeys <- &secp256k1.PrivKey{Key: privKeyBytes}
	}

	// Start Kava GRPC client
	encodingConfig := kava.MakeEncodingConfig()
	d.KavaClient = NewGrpcKavaClient(d.config.Kava.Endpoint, encodingConfig)

	// Set up Binance Chain client
	bncNetwork := btypes.TestNetwork
	if d.config.BinanceChain.ChainID == "Binance-Chain-Tigris" {
		bncNetwork = btypes.ProdNetwork
	}
	d.BnbClient = brpc.NewRPCClient(d.config.BinanceChain.Endpoint, bncNetwork)
	bnbKeyManager, err := bkeys.NewMnemonicKeyManager(d.config.BinanceChain.Mnemonic)
	if err != nil {
		panic(err)
	}
	d.BnbClient.SetKeyManager(bnbKeyManager)

	// Run Workers
	for {
		select {
		case <-ctx.Done():
			return
		case claim := <-d.jobQueue:
			logger := log.WithFields(log.Fields{
				"request_id":   claim.ID,
				"swap_id":      claim.SwapID,
				"target_chain": claim.TargetChain,
			})
			logger.Info("claim request begin processing")
			switch strings.ToUpper(claim.TargetChain) {
			case types.TargetKava:
				// fetch an available mnemonic, waiting if none available // TODO should respect ctx
				key := <-kavaKeys

				go func() {
					// release the mnemonic when done
					defer func() { kavaKeys <- key }()
					Retry(30, 20*time.Second, logger, func() (string, string, error) {
						return claimOnKava(d.config.Kava, d.KavaClient, claim, key)
					})
				}()
			case types.TargetBinance, types.TargetBinanceChain:
				// TODO make binance safe for concurrent requests
				go func() {
					Retry(30, 20*time.Second, logger, func() (string, string, error) {
						return claimOnBinanceChain(d.BnbClient, claim)
					})
				}()
			}
		}
	}
}

func (d *Dispatcher) JobQueue() chan types.ClaimJob { return d.jobQueue }
