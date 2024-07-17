package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/signing"
	"github.com/kava-labs/kava/app"
	"github.com/rs/zerolog"
)

func main() {
	ctx := context.Background()
	// create base logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

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
		logger.Fatal().Err(err).Send()
		os.Exit(1)
	}

	logger.
		Info().
		Str("chainId", config.KavaChainId).
		Str("grpcUrl", config.KavaGrpcUrl).
		Dur("bidInterval", config.KavaBidInterval).
		Str("profitMargin", config.ProfitMargin.String()).
		Msg("config loaded")

	//
	// create codec for messages
	//
	encodingConfig := app.MakeEncodingConfig()

	//
	// create rpc client for fetching data
	// required for bidding
	//
	logger.Info().Msg("creating grpc client")

	grpcClient := NewGrpcClient(config.KavaGrpcUrl, encodingConfig.Marshaler)
	defer grpcClient.GrpcClientConn.Close()

	//
	// client for broadcasting txs
	//
	hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, 0)
	privKeyBytes, err := hd.Secp256k1.Derive()(config.KavaKeeperMnemonic, "", hdPath.String())
	if err != nil {
		logger.Fatal().
			Err(err).
			Msg("failed to derive key")
	}
	// wrap with cosmos secp256k1 private key struct
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	logger.Info().
		Str("signing address", sdk.AccAddress(privKey.PubKey().Address()).String()).
		Send()

	signer, err := signing.NewSigner(
		config.KavaChainId,
		signing.EncodingConfigAdapter{EncodingConfig: encodingConfig},
		grpcClient.Auth,
		grpcClient.Tx,
		privKey,
		100,
		logger,
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to initialize signer")
	}

	startHealthCheckService(
		ctx,
		logger,
		config,
		grpcClient,
		signer,
	)

	// channels to communicate with signer
	requests := make(chan signing.MsgRequest)

	// signer starts it's own go routines and returns
	responses, err := signer.Run(ctx, requests)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to start signer")
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
			logger.Error().
				Err(err).
				Int("priceErrors", priceErrors).
				Msgf("failed to get auction data, retrying")

			priceErrors += 1
			logger.Debug().Err(err).Msg("failed to fetch auction data")
			time.Sleep(time.Second * 5)
			continue
		}

		logger.Info().Msgf("fetched prices after %d attempt(s)\n", priceErrors+1)
		priceErrors = 0

		// apply price overrides
		for denom, price := range config.PriceOverrides {
			info := data.Assets[denom]
			info.Price = price
			data.Assets[denom] = info
		}

		latestHeight, err := grpcClient.LatestHeight()
		if err != nil {
			continue
		}

		logger.Info().Msgf("latest height: %d", latestHeight)
		logger.Info().Msgf("checking %d auctions", len(data.Auctions))

		auctionBids := GetBids(
			logger,
			data,
			sdk.AccAddress(privKey.PubKey().Address()),
			config.ProfitMargin,
		)

		msgs := CreateBidMsgs(sdk.AccAddress(privKey.PubKey().Address()), auctionBids)
		logger.Info().Msgf("creating %d bids", len(msgs))

		totalBids := sdk.Coins{}
		for _, bid := range msgs {
			totalBids = totalBids.Add(bid.Amount)
		}
		logger.Info().Msgf("total for bids: %s", totalBids)

		auctionDups := make(map[uint64]int64)
		for _, bid := range msgs {
			auctionDups[bid.AuctionId] = auctionDups[bid.AuctionId] + 1
		}

		for auctionID, numDups := range auctionDups {
			if numDups > 1 {
				logger.Info().Msgf("auction id %d dups %d", auctionID, numDups)
			}
		}

		// gas limit of one bit
		gasBaseLimit := uint64(300000)

		// max gas price to get into any block
		gasPrice := 0.05

		// aggregator for msgs between loops
		msgBatch := []sdk.Msg{}
		// total number of messages
		numMsgs := len(msgs)

		for i, msg := range msgs {
			logger.Debug().Interface("bid msg", msg).Send()
			// collect msgs
			msgCopy := msg
			msgBatch = append(msgBatch, &msgCopy)

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
