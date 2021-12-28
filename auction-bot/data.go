package main

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	"github.com/kava-labs/kava/x/pricefeed/types"
	"google.golang.org/grpc/metadata"
)

type AssetInfo struct {
	Price            sdk.Dec
	ConversionFactor sdk.Int
}

type AuctionData struct {
	Assets       map[string]AssetInfo
	Auctions     []auctiontypes.Auction
	BidIncrement sdk.Dec
	BidMargin    sdk.Dec
}

func GetAuctionData(client GrpcClient, cdc codec.Codec) (*AuctionData, error) {
	// fetch latest block to get height
	latestHeight, err := client.LatestHeight()
	if err != nil {
		return nil, err
	}

	pricesRes, err := client.Pricefeed.Prices(ctxAtHeight(latestHeight), &types.QueryPricesRequest{})
	if err != nil {
		return nil, err
	}

	auctionsRes, err := client.Auction.Auctions(ctxAtHeight(latestHeight), &auctiontypes.QueryAuctionsRequest{})
	if err != nil {
		return nil, err
	}

	cdpParamsRes, err := client.Cdp.Params(ctxAtHeight(latestHeight), &cdptypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	hardParamsRes, err := client.Hard.Params(ctxAtHeight(latestHeight), &hardtypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	markets := deduplicateMarkets(cdpParamsRes.Params.CollateralParams, hardParamsRes.Params.MoneyMarkets)

	// map price data
	priceData := make(map[string]sdk.Dec)
	for _, price := range pricesRes.Prices {
		priceData[price.MarketID] = price.Price
	}

	// loop markets and create AssetInfo
	assetInfo := make(map[string]AssetInfo)
	for _, market := range markets {
		price, ok := priceData[market.SpotMarketID]
		if !ok {
			return nil, fmt.Errorf("no price for market id %s", market.SpotMarketID)
		}
		assetInfo[market.Denom] = AssetInfo{
			Price:            price,
			ConversionFactor: market.ConversionFactor,
		}
	}

	var auctions []auctiontypes.Auction
	for _, anyAuction := range auctionsRes.Auction {
		var auction auctiontypes.Auction
		cdc.UnpackAny(anyAuction, &auction)

		auctions = append(auctions, auction)
	}

	return &AuctionData{
		Assets:       assetInfo,
		Auctions:     auctions,
		BidIncrement: sdk.MustNewDecFromStr("0.01"), // TODO could fetch increment from chain
	}, nil
}

func ctxAtHeight(height int64) context.Context {
	heightStr := strconv.FormatInt(height, 10)
	return metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, heightStr)
}

func deduplicateMarkets(cdpMarkets cdptypes.CollateralParams, hardMarkets hardtypes.MoneyMarkets) []auctionMarket {
	seenDenoms := make(map[string]bool)

	markets := []auctionMarket{}

	for _, cdpMarket := range cdpMarkets {
		_, seen := seenDenoms[cdpMarket.Denom]
		if seen {
			continue
		}
		conversionFactor := cdpMarket.ConversionFactor
		i := big.NewInt(10)
		i.Exp(i, conversionFactor.BigInt(), nil)
		markets = append(markets, auctionMarket{cdpMarket.Denom, cdpMarket.SpotMarketID, sdk.NewIntFromBigInt(i)})
		seenDenoms[cdpMarket.Denom] = true
	}
	for _, hardMarket := range hardMarkets {
		_, seen := seenDenoms[hardMarket.Denom]
		if seen {
			continue
		}
		markets = append(markets, auctionMarket{hardMarket.Denom, hardMarket.SpotMarketID, hardMarket.ConversionFactor})
		seenDenoms[hardMarket.Denom] = true
	}
	return markets
}

type auctionMarket struct {
	Denom            string
	SpotMarketID     string
	ConversionFactor sdk.Int
}
