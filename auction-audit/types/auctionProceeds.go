package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuctionIDToHeightMap maps auction ID -> block height
type AuctionIDToHeightMap map[uint64]int64

// AuctionProceeds defines additional information about a single auction
type AuctionProceeds struct {
	BaseAuctionProceeds

	UsdValueBefore sdk.Dec
	UsdValueAfter  sdk.Dec
	PercentLoss    sdk.Dec
}

// BaseAuctionProceeds defines basic information about a single auction
type BaseAuctionProceeds struct {
	ID                uint64
	AmountPurchased   sdk.Coin
	AmountPaid        sdk.Coin
	InitialLot        sdk.Coin
	LiquidatedAccount string
	WinningBidder     string
	SourceModule      string
}

// BaseAuctionProceedsMap maps auction ID -> BaseAuctionProceeds
type BaseAuctionProceedsMap map[uint64]BaseAuctionProceeds

// AuctionProceedsMap maps auction ID -> AuctionProceeds
type AuctionProceedsMap map[uint64]AuctionProceeds

// ToRecords converts AuctionProceedsMap to a 2D string array
func (ap AuctionProceedsMap) ToRecords() [][]string {
	var records [][]string

	for _, ap := range ap {
		records = append(records, []string{
			DenomMap[ap.AmountPurchased.Denom],
			ap.AmountPurchased.Amount.ToDec().Mul(sdk.OneDec().Quo(ConversionMap[ap.AmountPurchased.Denom].ToDec())).String(),
			DenomMap[ap.AmountPaid.Denom],
			ap.AmountPaid.Amount.ToDec().Mul(sdk.OneDec().Quo(ConversionMap[ap.AmountPaid.Denom].ToDec())).String(),
			ap.InitialLot.String(),
			ap.LiquidatedAccount,
			ap.WinningBidder,
			ap.UsdValueBefore.String(),
			ap.UsdValueAfter.String(),
			ap.PercentLoss.String(),
		})
	}

	return records
}
