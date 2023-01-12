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
		"bidder", config.BidderAddress.String(),
		"start height", config.StartHeight,
		"end height", config.EndHeight,
	).Info("config loaded")

	//
	// create codec for messages
	//
	encodingConfig := app.MakeEncodingConfig()

	//
	// create grpc client and test that it's responding
	grpcClient := NewGrpcClient(config.GrpcURL, encodingConfig.Marshaler, encodingConfig.TxConfig)
	defer grpcClient.GrpcClientConn.Close()
	nodeInfoResponse, err := grpcClient.Tm.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	if err != nil {
		return fmt.Errorf("failed to fetch chain id: %w", err)
	}

	logger.Info(fmt.Sprintf("chain id: %s", nodeInfoResponse.DefaultNodeInfo.Network))

	//
	// crawl blocks to find auctions and inbound transfers
	//
	auctionIdToHeightMap, err := GetAuctionEndData(
		grpcClient,
		config.StartHeight,
		config.EndHeight,
		config.BidderAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch auction end data: %w", err)
	}
	fmt.Printf("Found %d auctions\n", len(auctionIdToHeightMap))
	fmt.Printf("Auction end data: %v \n", auctionIdToHeightMap)

	//
	// fetch the final clearing data for auctions that the bidder address won
	//
	auctionClearingMap, err := GetAuctionClearingData(
		grpcClient,
		auctionIdToHeightMap,
		config.BidderAddress,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch auction clearing data: %w", err)
	}

	// Add additional information to auction data map including USD value before and after liquidation
	fullAuctionDataMap, err := GetAuctionValueData(context.Background(), grpcClient, auctionClearingMap)
	if err != nil {
		return fmt.Errorf("failed to fetch source collateral data: %w", err)
	}

	outputFile, err := csv.GetFileOutput("auction_summary", config)
	if err != nil {
		return fmt.Errorf("failed to get file output: %w", err)
	}

	err = csv.WriteCsv(
		outputFile,
		[]string{
			"Asset Purchased",
			"Amount Purchased",
			"Asset Paid",
			"Amount Paid",
			"Initial Lot",
			"Liquidated Account",
			"Winning Bidder Account",
			"USD Value Before Liquidation",
			"USD Value After Liquidation",
			"Percent Loss",
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
