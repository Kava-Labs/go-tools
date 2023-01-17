package types

import (
	"fmt"
	"sort"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuctionIDToHeightMap maps auction ID -> block height
type AuctionIDToHeightMap map[uint64]int64

// AuctionProceeds defines additional information about a single auction
type AuctionProceeds struct {
	BaseAuctionProceeds

	InitialLot          sdk.Coin
	USDValueBefore      sdk.Dec
	USDValueAfter       sdk.Dec
	AmountReturned      sdk.Coin
	PercentLossAmount   sdk.Dec
	PercentLossUSDValue sdk.Dec
}

// BaseAuctionProceeds defines basic information about a single auction
type BaseAuctionProceeds struct {
	ID                uint64
	EndHeight         int64
	AmountPurchased   sdk.Coin
	AmountPaid        sdk.Coin
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
			fmt.Sprintf("%d", ap.ID),
			fmt.Sprintf("%d", ap.EndHeight),
			DenomMap[ap.AmountPurchased.Denom],
			ap.AmountPurchased.Amount.ToDec().
				Quo(ConversionMap[ap.AmountPurchased.Denom].ToDec()).
				String(),
			DenomMap[ap.AmountPaid.Denom],
			ap.AmountPaid.Amount.ToDec().
				Quo(ConversionMap[ap.AmountPaid.Denom].ToDec()).
				String(),
			ap.InitialLot.Amount.ToDec().
				Quo(ConversionMap[ap.InitialLot.Denom].ToDec()).
				String(),
			ap.LiquidatedAccount,
			ap.WinningBidder,
			ap.USDValueBefore.String(),
			ap.USDValueAfter.String(),
			ap.AmountReturned.Amount.ToDec().
				Quo(ConversionMap[ap.AmountReturned.Denom].ToDec()).
				String(),
			ap.PercentLossAmount.String(),
			ap.PercentLossUSDValue.String(),
		})
	}

	// Sort output by auction ID
	sort.Slice(records, func(i, j int) bool {
		return MustStrToInt(records[i][0]) < MustStrToInt(records[j][0])
	})

	return records
}

func MustStrToInt(str string) int64 {
	i, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		panic(err)
	}

	return i
}
