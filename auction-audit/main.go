package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/kava-labs/kava/app"
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
		logger.Error("failed to fetch chain id")
		logger.Error(err.Error())
		os.Exit(1)
	}
	logger.Info(fmt.Sprintf("chain id: %s", nodeInfoResponse.DefaultNodeInfo.Network))

	//
	// crawl blocks to find auctions and inbound transfers
	//
	auctionEndMap, transferMap, err := GetAuctionEndData(grpcClient, config.StartHeight, config.EndHeight, config.BidderAddress)
	if err != nil {
		logger.Error("failed to fetch auction end data")
		logger.Error(err.Error())
		os.Exit(1)
	}
	fmt.Printf("Found %d auctions\n", len(auctionEndMap))

	//
	// fetch the final clearing data for auctions that the bidder address won
	//
	auctionClearingMap, err := GetAuctionClearingData(grpcClient, auctionEndMap, config.BidderAddress)
	if err != nil {
		logger.Error("failed to fetch auction clearing data")
		logger.Error(err.Error())
		os.Exit(1)
	}

	// write auction results to file
	csvFile, err := os.Create("auction_summary_20220612_20220620.csv")
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer csvFile.Close()

	w := csv.NewWriter(csvFile)
	defer w.Flush()
	err = w.Write([]string{"Asset Purchased", "Amount Purchase", "Asset Paid", "Amount Paid"})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	for _, apMap := range auctionClearingMap {
		for _, ap := range apMap {
			row := []string{
				denomMap[ap.AmountPurchased.Denom],
				ap.AmountPurchased.Amount.ToDec().Mul(sdk.OneDec().Quo(conversionMap[ap.AmountPurchased.Denom].ToDec())).String(),
				denomMap[ap.AmountPaid.Denom],
				ap.AmountPaid.Amount.ToDec().Mul(sdk.OneDec().Quo(conversionMap[ap.AmountPaid.Denom].ToDec())).String()}
			err := w.Write(row)
			if err != nil {
				logger.Error(err.Error())
				os.Exit(1)
			}
		}
	}

	// write in-bound transfers to file
	csvFile, err = os.Create("auction_transfers_20220612_20220620.csv")
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
				denomMap[coin.Denom],
				coin.Amount.ToDec().Mul(sdk.OneDec().Quo(conversionMap[coin.Denom].ToDec())).String(),
			}
			err := w.Write(row)
			if err != nil {
				logger.Error(err.Error())
				os.Exit(1)
			}
		}
	}

}

var denomMap = map[string]string{
	"usdx":  "USDX",
	"bnb":   "BNB",
	"btcb":  "BTC",
	"hard":  "HARD",
	"ukava": "KAVA",
	"xrpb":  "XRP",
	"busd":  "BUSD",
	"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2": "ATOM",
	"swp": "SWP",
	"ibc/799FDD409719A1122586A629AE8FCA17380351A51C1F47A80A1B8E7F2A491098": "AKT",
	"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C": "UST",
}

var conversionMap = map[string]sdk.Int{
	"usdx":  sdk.NewInt(1000000),
	"bnb":   sdk.NewInt(100000000),
	"btcb":  sdk.NewInt(100000000),
	"hard":  sdk.NewInt(1000000),
	"ukava": sdk.NewInt(1000000),
	"xrpb":  sdk.NewInt(100000000),
	"busd":  sdk.NewInt(100000000),
	"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2": sdk.NewInt(1000000),
	"swp": sdk.NewInt(1000000),
	"ibc/799FDD409719A1122586A629AE8FCA17380351A51C1F47A80A1B8E7F2A491098": sdk.NewInt(1000000),
	"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C": sdk.NewInt(1000000),
}
