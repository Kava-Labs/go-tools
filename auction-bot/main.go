package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/kava-labs/go-tools/signing"
	"github.com/kava-labs/kava/app"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	kavaGrpcUrlEnvKey = "KAVA_GRPC_URL"
	mnemonicEnvKey    = "KEEPER_MNEMONIC"
	profitMarginKey   = "BID_MARGIN"
	bidIntervalKey    = "BID_INTERVAL"
)

func main() {
	// create base logger
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	//
	// bootstrap kava chain config
	//
	// sets a global cosmos sdk for bech32 prefix
	//
	// required before loading config
	//
	app.SetSDKConfig()

	//
	// Load config
	//
	// if config is not valid, exit with fatal error
	//
	config, err := LoadConfig(&EnvLoader{})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.With(
		"grpcUrl", config.KavaGrpcUrl,
		"bidInterval", config.KavaBidInterval.String(),
		"profitMargin", config.ProfitMargin.String(),
	).Info("config loaded")

	//
	// create codec for messages
	//
	encodingConfig := app.MakeEncodingConfig()

	//
	// create rpc client for fetching data
	// required for bidding
	//
	logger.Info("creating grpc client")

	grpcClient := NewGrpcClient(config.KavaGrpcUrl, encodingConfig.Marshaler)
	defer grpcClient.GrpcClientConn.Close()

	//
	// client for broadcasting txs
	//
	hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, 0)
	privKeyBytes, err := hd.Secp256k1.Derive()(config.KavaKeeperMnemonic, "", hdPath.String())
	if err != nil {
		logger.Error("failed to derive key")
		logger.Error(err.Error())
		os.Exit(1)
	}
	// wrap with cosmos secp256k1 private key struct
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	logger.Info(fmt.Sprintf("signing address: %s", sdk.AccAddress(privKey.PubKey().Address()).String()))

	nodeInfoResponse, err := grpcClient.Tm.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	if err != nil {
		logger.Error("failed to fetch chain id")
		logger.Error(err.Error())
		os.Exit(1)
	}
	signer := signing.NewSigner(
		nodeInfoResponse.DefaultNodeInfo.Network,
		encodingConfig,
		grpcClient.Auth,
		grpcClient.Tx,
		privKey,
		100,
	)

	// channels to communicate with signer
	requests := make(chan signing.MsgRequest)

	// signer starts it's own go routines and returns
	responses, err := signer.Run(requests)
	if err != nil {
		logger.Error("failed to start signer")
		logger.Error(err.Error())
		os.Exit(1)
	}

	// log responses, if responses are not read, requests will block
	go func() {
		for {
			// response is not returned until the msg is committed to a block
			response := <-responses

			// error will be set if response is not Code 0 (success) or Code 19 (already in mempool)
			if response.Err != nil {
				fmt.Printf("response code: %d error %s\n", response.Result.Code, response.Err)
				continue
			}

			// code and result are from broadcast, not deliver tx
			// it is up to the caller/requester to check the deliver tx code and deal with failure
			fmt.Printf("response code: %d, hash %s\n", response.Result.Code, response.Result.TxHash)
		}
	}()

	priceErrors := 0
	for {
		data, err := GetAuctionData(grpcClient, encodingConfig.Marshaler)
		if err != nil {
			priceErrors += 1
			continue
		}
		logger.Info(fmt.Sprintf("fetched prices after %d attempt(s)\n", priceErrors+1))
		priceErrors = 0

		latestHeight, err := grpcClient.LatestHeight()

		if err != nil {
			continue
		}

		logger.Info(fmt.Sprintf("latest height: %d", latestHeight))
		logger.Info(fmt.Sprintf("checking %d auctions", len(data.Auctions)))

		auctionBids := GetBids(data, sdk.AccAddress(privKey.PubKey().Address()), config.ProfitMargin)

		msgs := CreateBidMsgs(sdk.AccAddress(privKey.PubKey().Address()), auctionBids)
		logger.Info(fmt.Sprintf("creating %d bids", len(msgs)))

		totalBids := sdk.Coins{}
		for _, bid := range msgs {
			totalBids = totalBids.Add(bid.Amount)

		}
		logger.Info(fmt.Sprintf("total usdx for bids %s", totalBids))

		auctionDups := make(map[uint64]int64)
		for _, bid := range msgs {
			auctionDups[bid.AuctionId] = auctionDups[bid.AuctionId] + 1
		}

		for auctionID, numDups := range auctionDups {
			logger.Info(fmt.Sprintf("auction id %d dups %d", auctionID, numDups))
		}

		// gas limit of one bit
		gasBaseLimit := uint64(300000)

		// max gas price to get into any block
		gasPrice := 0.25

		// aggregator for msgs between loops
		msgBatch := []sdk.Msg{}
		// total number of messages
		numMsgs := len(msgs)

		for i, msg := range msgs {
			// collect msgs
			msgBatch = append(msgBatch, &msg)

			// when batch is 10 or on the last loop, send request
			if len(msgBatch) == 10 || i == numMsgs-1 {
				// batch size may be less for last loop
				batchSize := len(msgBatch)

				// copy slice to avoid slice re-use
				requestMsgBatch := make([]sdk.Msg, batchSize)
				copy(requestMsgBatch, msgBatch)
				// reset batch
				msgBatch = []sdk.Msg{}

				// add up total gas for tx
				gasLimit := gasBaseLimit * uint64(batchSize)
				// calculate gas fee amount, ceil so we stay over limit on rounding
				feeAmount := int64(math.Ceil(float64(gasLimit) * gasPrice))

				requests <- signing.MsgRequest{
					Msgs:      requestMsgBatch,
					GasLimit:  gasLimit,
					FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(feeAmount))),
					Memo:      "",
				}
			}
		}

		// wait for next interval
		time.Sleep(config.KavaBidInterval)
	}
}
