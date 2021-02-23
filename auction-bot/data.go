package main

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
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
		return nil, err
	}

	auctions, err := client.GetAuctions(height)
	if err != nil {
		return nil, err
	}

	markets, err := client.GetMarkets(height)
	if err != nil {
		return nil, err
	}

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
		conversionFactor := market.ConversionFactor
		i := big.NewInt(10)
		i.Exp(i, conversionFactor.BigInt(), nil)

		assetInfo[market.Denom] = AssetInfo{
			Price:            price,
			ConversionFactor: sdk.NewIntFromBigInt(i),
		}
	}

	usdxInfo := AssetInfo{
		Price:            sdk.OneDec(),
		ConversionFactor: sdk.NewInt(1000000),
	}
	assetInfo["usdx"] = usdxInfo

	return &AuctionData{
		Assets:       assetInfo,
		Auctions:     auctions,
		BidIncrement: sdk.MustNewDecFromStr("0.01"), // TODO could fetch increment from chain
	}, nil
}
