package auctions

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuctionInfo contains information about an auction
type AuctionInfo struct {
	ID     uint64
	Bidder sdk.AccAddress
	Amount sdk.Coin
}

// AuctionInfos is an array of AuctionInfo
type AuctionInfos []AuctionInfo

func CalculateTotalAuctionsUSDValue(data *AuctionData) (sdk.Dec, error) {
	totalValue := sdk.NewDec(0)

	for _, auction := range data.Auctions {
		lot := auction.GetLot()
		assetInfo, ok := data.Assets[lot.Denom]
		if !ok {
			return sdk.Dec{}, fmt.Errorf("missing asset info for %s", lot.Denom)
		}

		usdValue := CalculateUSDValue(lot, assetInfo)
		totalValue = totalValue.Add(usdValue)
	}

	return totalValue, nil
}

// calculateUSDValue calculates the USD value of a given Coin and AssetInfo
func CalculateUSDValue(coin sdk.Coin, assetInfo AssetInfo) sdk.Dec {
	return coin.Amount.ToDec().Quo(assetInfo.ConversionFactor.ToDec()).Mul(assetInfo.Price)
}
