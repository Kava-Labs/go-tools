package auctions

import (
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

// calculateUSDValue calculates the USD value of a given Coin and AssetInfo
func CalculateUSDValue(coin sdk.Coin, assetInfo AssetInfo) sdk.Dec {
	return coin.Amount.ToDec().Quo(assetInfo.ConversionFactor.ToDec()).Mul(assetInfo.Price)
}
