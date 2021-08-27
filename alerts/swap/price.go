package swap

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var denomRewrites = map[string]string{
	"btcb":  "btc",
	"xrpb":  "xrp",
	"ukava": "kava",
}

// GetDenomRewrite returns rewritten denoms for certain assets that may be
// difficult to find the actual USD value of, such as BTCV
func GetDenomRewrite(denom string) string {
	rewritten, ok := denomRewrites[denom]
	if !ok {
		return denom
	}

	return rewritten
}

// GetUsdPrice returns the USD value of a given denom
func GetUsdPrice(denom string, data SwapPoolsData) (sdk.Dec, error) {
	denom = GetDenomRewrite(denom)

	// First check if the price exists in the pricefeed module
	usd, err := data.PricefeedPrices.UsdValue(denom)
	if err == nil {
		return usd, nil
	}

	// If it doesn't exist in pricefeed, look it up in binance and convert to USD
	usd, err = data.BinancePrices.UsdValue(denom, data.UsdValues.Busd)
	if err != nil {
		return sdk.Dec{}, err
	}

	return usd, nil
}
