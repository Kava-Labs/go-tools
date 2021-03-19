package main

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
)

type AssetInfo struct {
	Price            sdk.Dec
	ConversionFactor sdk.Int
}

type AuctionData struct {
	Assets       map[string]AssetInfo
	Auctions     auctiontypes.Auctions
	BidIncrement sdk.Dec
	BidMargin    sdk.Dec
}

func GetAuctionData(client AuctionClient) (*AuctionData, error) {
	// fetch chain info to get height
	info, err := client.GetInfo()
	if err != nil {
		return nil, err
	}

	// use height to get consistent state from rpc client
	height := info.LatestHeight

	fmt.Printf("latest height: %d\n", info.LatestHeight)

	prices, err := client.GetPrices(height)
	if err != nil {
		fmt.Printf("Errored querying market prices: %v\n", err)
		return nil, err
	}

	auctions, err := client.GetAuctions(height)
	if err != nil {
		return nil, err
	}

	cdpMarkets, err := client.GetMarkets(height)
	if err != nil {
		return nil, err
	}
	markets := filterMarkets(cdpMarkets)
	// map price data
	priceData := make(map[string]sdk.Dec)
	for _, price := range prices {
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

	return &AuctionData{
		Assets:       assetInfo,
		Auctions:     auctions,
		BidIncrement: sdk.MustNewDecFromStr("0.01"), // TODO could fetch increment from chain
	}, nil
}

func filterMarkets(cdpMarkets cdptypes.CollateralParams) []auctionMarket {
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
	usdxMarket := auctionMarket{Denom: "usdx", SpotMarketID: "usdx:usd", ConversionFactor: sdk.NewInt(1000000)}
	markets = append(markets, usdxMarket)
	return markets
}

type auctionMarket struct {
	Denom            string
	SpotMarketID     string
	ConversionFactor sdk.Int
}
