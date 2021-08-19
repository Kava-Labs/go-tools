package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AuctionInfo struct {
	ID     uint64
	Bidder sdk.AccAddress
	Amount sdk.Coin
}

type AuctionInfos []AuctionInfo

func calculateUSDValue(coin sdk.Coin, assetInfo AssetInfo) sdk.Dec {
	return coin.Amount.ToDec().Quo(assetInfo.ConversionFactor.ToDec()).Mul(assetInfo.Price)
}
