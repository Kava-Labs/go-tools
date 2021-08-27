package swap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	BinancePriceApiEndpoint = "https://api.binance.com/api/v3/ticker/price"
	BusdDenom               = "BUSD"
)

// BinancePrices maps the symbol to price
type BinancePrices map[string]sdk.Dec

type BinancePrice struct {
	Symbol string  `json:"symbol"`
	Price  sdk.Dec `json:"price"`
}

func GetBinancePrices() (BinancePrices, error) {
	resp, err := http.Get(BinancePriceApiEndpoint)
	if err != nil {
		return BinancePrices{}, err
	}

	defer resp.Body.Close()

	// Deserialize as an array first
	var pricesArr []BinancePrice
	if err := json.NewDecoder(resp.Body).Decode(&pricesArr); err != nil {
		return BinancePrices{}, err
	}

	// Create a map with previous array
	prices := make(map[string]sdk.Dec)
	for _, v := range pricesArr {
		prices[v.Symbol] = v.Price
	}

	return prices, nil
}

// BusdValue returns the BUSD value of a given denom
func (prices BinancePrices) BusdValue(denom string) (sdk.Dec, error) {
	price, ok := prices[strings.ToUpper(denom)+BusdDenom]
	if ok {
		return price, nil
	}

	// Try reversed symbol
	price, ok = prices[BusdDenom+strings.ToUpper(denom)]
	if !ok {
		return sdk.Dec{}, fmt.Errorf("Failed to find price for %v", denom)
	}

	return price, nil
}

// UsdValue returns the USD value of a given denom and USD price of BUSD
func (prices BinancePrices) UsdValue(denom string, BusdUsdPrice sdk.Dec) (sdk.Dec, error) {
	priceInBusd, err := prices.BusdValue(denom)
	if err != nil {
		return sdk.Dec{}, err
	}

	return priceInBusd.Mul(BusdUsdPrice), nil
}
