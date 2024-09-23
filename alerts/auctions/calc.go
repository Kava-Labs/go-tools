package auctions

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
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

// CalculateUSDValue calculates the USD value of a given Coin and AssetInfo
func CalculateUSDValue(coin sdk.Coin, assetInfo AssetInfo) sdk.Dec {
	return coin.Amount.ToLegacyDec().Quo(assetInfo.ConversionFactor.ToLegacyDec()).Mul(assetInfo.Price)
}

func CheckInefficientAuctions(data *AuctionData, thresholdUSD, thresholdRatio sdk.Dec, thresholdTime time.Duration) ([]auctiontypes.Auction, error) {
	inefficientAuctions := []auctiontypes.Auction{}
	for _, auction := range data.Auctions {
		// check if the time remaining on the auction is below the threshold
		endTime := auction.GetEndTime()
		timeRemaining := time.Until(endTime)
		if timeRemaining > thresholdTime {
			continue
		}

		// check that the lot USD value is above the threshold
		lot := auction.GetLot()
		lotAssetInfo, ok := data.Assets[lot.Denom]
		if !ok {
			return []auctiontypes.Auction{}, fmt.Errorf("missing asset info for %s", lot.Denom)
		}
		usdValueLot := CalculateUSDValue(lot, lotAssetInfo)
		if usdValueLot.LT(thresholdUSD) {
			continue
		}

		// check that the bidUSD:lotUSD ratio is below the threshold
		bid := auction.GetBid()
		bidAssetInfo, ok := data.Assets[bid.Denom]
		if !ok {
			return []auctiontypes.Auction{}, fmt.Errorf("missing asset info for %s", bid.Denom)
		}
		usdValueBid := CalculateUSDValue(bid, bidAssetInfo)
		ratio := usdValueBid.Quo(usdValueLot)
		if ratio.GTE(thresholdRatio) {
			continue
		}
		inefficientAuctions = append(inefficientAuctions, auction)

	}
	return inefficientAuctions, nil
}
