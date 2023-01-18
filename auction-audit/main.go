package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/kava-labs/go-tools/auction-audit/config"
	"github.com/kava-labs/go-tools/auction-audit/csv"
	"github.com/kava-labs/kava/app"
)

func tryMain(logger log.Logger) error {

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
	config, err := config.LoadConfig(&config.EnvLoader{})
	if err != nil {
		return err
	}

	logger.With(
		"grpcUrl", config.GrpcURL,
		"rpcUrl", config.RpcURL,
		"start height", config.StartHeight,
		"end height", config.EndHeight,
	).Info("config loaded")

	//
	// create codec for messages
	//
	encodingConfig := app.MakeEncodingConfig()

	//
	// create grpc client and test that it's responding
	grpcClient, err := NewGrpcClient(
		config.GrpcURL,
		config.RpcURL,
		encodingConfig.Marshaler,
		encodingConfig.TxConfig,
	)
	if err != nil {
		return err
	}

	defer grpcClient.GrpcClientConn.Close()
	nodeInfoResponse, err := grpcClient.Tm.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch chain id: %w", err)
	}

	logger.Info(fmt.Sprintf("chain id: %s", nodeInfoResponse.DefaultNodeInfo.Network))

	// Crawl blocks to find auctions and inbound transfers
	logger.Info("Fetching auction end data... this may take a while")
	auctionIdToHeightMap, err := GetAuctionEndData(
		logger,
		grpcClient,
		config.StartHeight,
		config.EndHeight,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch auction end data: %w", err)
	}
	logger.Info("Found auctions", "count", len(auctionIdToHeightMap))

	if len(auctionIdToHeightMap) == 0 {
		logger.Info("No auctions found, stopping.")
		return nil
	}

	// Fetch the final clearing data for auctions that the bidder address won
	logger.Info("Fetching auction clearing data (auction winners)...")
	auctionClearingMap, err := GetAuctionClearingData(
		logger,
		grpcClient,
		auctionIdToHeightMap,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch auction clearing data: %w", err)
	}

	// Add additional information to auction data map including USD value before and after liquidation
	logger.Info("Fetching auction source USD value data...")
	fullAuctionDataMap, err := GetAuctionValueData(
		context.Background(),
		logger,
		grpcClient,
		auctionClearingMap,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch source collateral data: %w", err)
	}

	outputFile, err := csv.GetFileOutput("auction_summary", config)
	if err != nil {
		return fmt.Errorf("failed to get file output: %w", err)
	}

	logger.Info(
		"Writing output data to csv",
		"fileName", outputFile.Name(),
	)

	err = csv.WriteCsv(
		outputFile,
		[]string{
			"Auction ID",
			"End Height",
			"Source Module",
			"Asset Purchased",
			"Amount Purchased",
			"Asset Paid",
			"Amount Paid",
			"Initial Lot",
			"Liquidated Account",
			"Winning Bidder Account",
			"USD Value Before Liquidation",
			"USD Value After Liquidation",
			"Amount Returned",
			"Percent Loss (quantity)",
			"Percent Loss (USD value)",
		},
		fullAuctionDataMap,
	)
	if err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	return nil
}

func main() {
	// create base logger
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	if err := tryMain(logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
