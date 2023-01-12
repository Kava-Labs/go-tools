package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/kava-labs/go-tools/auction-audit/types"
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
	config, err := LoadConfig(&EnvLoader{})
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
	auctionIdToHeightMap, transferMap, err := GetAuctionEndData(
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

	fullAuctionDataMap, err := GetAuctionValueData(context.Background(), grpcClient, auctionClearingMap)
	if err != nil {
		return fmt.Errorf("failed to fetch source collateral data: %w", err)
	}

	var records [][]string

	for _, ap := range fullAuctionDataMap {
		records = append(records, []string{
			types.DenomMap[ap.AmountPurchased.Denom],
			ap.AmountPurchased.Amount.ToDec().Mul(sdk.OneDec().Quo(types.ConversionMap[ap.AmountPurchased.Denom].ToDec())).String(),
			types.DenomMap[ap.AmountPaid.Denom],
			ap.AmountPaid.Amount.ToDec().Mul(sdk.OneDec().Quo(types.ConversionMap[ap.AmountPaid.Denom].ToDec())).String(),
			ap.InitialLot.String(),
			ap.LiquidatedAccount,
			ap.WinningBidder,
			ap.UsdValueBefore.String(),
			ap.UsdValueAfter.String(),
			ap.PercentLoss.String(),
		})
	}

	outputFile, err := GetFileOutput("auction_summary", config)
	if err != nil {
		return fmt.Errorf("failed to get file output: %w", err)
	}

	err = WriteCsv(
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
		records,
	)
	if err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	// Fetch initial assets pre-liquidation, from CDP or HARD

	// write auction results to file
	csvFile, err := os.Create("auction_summary_20221215_20221217.csv")
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer csvFile.Close()

	w := csv.NewWriter(csvFile)
	defer w.Flush()
	err = w.Write([]string{
		"Asset Purchased",
		"Amount Purchased",
		"Asset Paid",
		"Amount Paid",
		"Initial Lot",
		"Liquidated Account",
		"Winning Bidder Account",
	})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	for _, ap := range auctionClearingMap {
		row := []string{
			types.DenomMap[ap.AmountPurchased.Denom],
			ap.AmountPurchased.Amount.ToDec().Mul(sdk.OneDec().Quo(types.ConversionMap[ap.AmountPurchased.Denom].ToDec())).String(),
			types.DenomMap[ap.AmountPaid.Denom],
			ap.AmountPaid.Amount.ToDec().Mul(sdk.OneDec().Quo(types.ConversionMap[ap.AmountPaid.Denom].ToDec())).String(),
			ap.InitialLot.String(),
			ap.LiquidatedAccount,
			ap.WinningBidder,
		}
		err := w.Write(row)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	}

	// write in-bound transfers to file
	csvFile, err = os.Create("auction_transfers_20221215_20221217.csv")
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer csvFile.Close()

	w = csv.NewWriter(csvFile)
	defer w.Flush()
	err = w.Write([]string{"Sender Address", "Sender Asset", "Sent Amount"})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	for sender, amount := range transferMap {
		for _, coin := range amount {
			row := []string{
				sender,
				types.DenomMap[coin.Denom],
				coin.Amount.ToDec().Mul(sdk.OneDec().Quo(types.ConversionMap[coin.Denom].ToDec())).String(),
			}
			err := w.Write(row)
			if err != nil {
				logger.Error(err.Error())
				os.Exit(1)
			}
		}
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
