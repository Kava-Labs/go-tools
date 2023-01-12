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
