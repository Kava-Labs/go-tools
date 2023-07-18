package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var DenomMap = map[string]string{
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

var DenomToPriceMarketMap = map[string]string{
	"usdx":  "usdx",
	"bnb":   "bnb",
	"btcb":  "btc",
	"hard":  "hard",
	"ukava": "kava",
	"xrpb":  "xrp",
	"busd":  "busd",
	"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2": "atom",
	"swp": "swp",
	"ibc/799FDD409719A1122586A629AE8FCA17380351A51C1F47A80A1B8E7F2A491098": "akt",
	"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C": "ust",
}

type PriceType int

const (
	PriceType_Spot PriceType = iota + 1
	PriceType_Twap30
)

func GetMarketID(denom string, priceType PriceType) (string, error) {
	market_name, found := DenomToPriceMarketMap[denom]
	if !found {
		return "", fmt.Errorf("could not find market id for denom %s", denom)
	}

	if priceType == PriceType_Spot {
		return fmt.Sprintf("%s:usd", market_name), nil
	}

	if priceType == PriceType_Twap30 {
		return fmt.Sprintf("%s:usd:30", market_name), nil
	}

	return "", fmt.Errorf("invalid priceType")
}

var ConversionMap = map[string]sdk.Int{
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
