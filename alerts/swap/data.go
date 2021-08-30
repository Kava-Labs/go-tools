package swap

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	swaptypes "github.com/kava-labs/kava/x/swap/types"
)

const (
	UsdxUsdMarketId = "usdx:usd"
	BusdUsdMarketId = "busd:usd"
	UsdDenom        = "usd"
)

// PricefeedPrices is a map of market_id to price corresponding to the Kava pricefeed module
type PricefeedPrices map[string]sdk.Dec

// UsdValue returns the USD value of a given denom via the pricefeed module
func (assets PricefeedPrices) UsdValue(denom string) (sdk.Dec, error) {
	price, ok := assets[strings.ToLower(denom+":"+UsdDenom)]
	if ok {
		// Found in pricefeed
		return price, nil
	}

	// Try reversed symbol
	price, ok = assets[strings.ToLower(UsdDenom+":"+denom)]
	if !ok {
		return sdk.Dec{}, fmt.Errorf("Failed to find price for %v", denom)
	}

	return price, nil
}

// UsdValues contains the market USD value of USDX and BUSD
type UsdValues struct {
	Usdx sdk.Dec
	Busd sdk.Dec
}

// cdpMarketMap is a map of denom to CollateralParam
type cdpMarketMap map[string]sdk.Int

// SwapPoolsData defines a map of AssetInfo and array of pools
type SwapPoolsData struct {
	PricefeedPrices PricefeedPrices
	UsdValues       UsdValues
	Pools           swaptypes.PoolStatsQueryResults
	BinancePrices   BinancePrices
	CdpMarkets      cdpMarketMap
}

// GetPoolsData returns current swap pools
func GetPoolsData(client SwapClient) (SwapPoolsData, error) {
	// fetch chain info to get height
	info, err := client.GetInfo()
	if err != nil {
		return SwapPoolsData{}, err
	}

	// use height to get consistent state from rpc client
	height := info.LatestHeight

	pools, err := client.GetPools(height)
	if err != nil {
		return SwapPoolsData{}, err
	}

	cdpMarkets, err := client.GetMarkets(height)
	if err != nil {
		return SwapPoolsData{}, err
	}
	// Add conversion factor for each of the collateral params
	cdpMarketData := make(map[string]sdk.Int)
	for _, market := range cdpMarkets {
		cdpMarketData[market.Denom] = market.ConversionFactor
	}

	debtParam, err := client.GetDebtParam(height)
	if err != nil {
		return SwapPoolsData{}, err
	}
	// Add USDX conversion factor
	cdpMarketData[debtParam.Denom] = debtParam.ConversionFactor

	prices, err := client.GetPrices(height)
	if err != nil {
		return SwapPoolsData{}, err
	}

	// map price data
	priceData := make(map[string]sdk.Dec)
	for _, price := range prices {
		priceData[price.MarketID] = price.Price
	}

	usdXValue, ok := priceData[UsdxUsdMarketId]
	if !ok {
		return SwapPoolsData{}, fmt.Errorf("Failed to price of %v", UsdxUsdMarketId)
	}

	busdValue, ok := priceData[BusdUsdMarketId]
	if !ok {
		return SwapPoolsData{}, fmt.Errorf("Failed to price of %v", BusdUsdMarketId)
	}

	binancePrices, err := GetBinancePrices()
	if err != nil {
		return SwapPoolsData{}, fmt.Errorf("Failed to fetch prices from Binance API %v", err.Error())
	}

	return SwapPoolsData{
		PricefeedPrices: priceData,
		UsdValues: UsdValues{
			Usdx: usdXValue,
			Busd: busdValue,
		},
		Pools:         pools,
		BinancePrices: binancePrices,
		CdpMarkets:    cdpMarketData,
	}, nil
}
