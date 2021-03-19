package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
)

type AuctionInfo struct {
	ID     uint64
	Bidder sdk.AccAddress
	Amount sdk.Coin
}

type AuctionInfos []AuctionInfo

func GetBids(data *AuctionData, keeper sdk.AccAddress, margin sdk.Dec) AuctionInfos {
	var auctionBidInfos AuctionInfos
	fmt.Printf("Checking %d auctions:\n", len(data.Auctions))
	for _, auction := range data.Auctions {
		switch auction.GetType() {
		case auctiontypes.CollateralAuctionType:
			switch auction.GetPhase() {
			case auctiontypes.ForwardAuctionPhase:
				bidInfo, shouldBid := handleForwardCollateralAuction(auction, keeper, data.Assets, margin)
				if !shouldBid {
					continue
				}
				auctionBidInfos = append(auctionBidInfos, bidInfo)
			case auctiontypes.ReverseAuctionPhase:
				bidInfo, shouldBid := handleReverseCollateralAuction(auction, keeper, data.Assets, data.BidIncrement, margin)
				if !shouldBid {
					continue
				}
				auctionBidInfos = append(auctionBidInfos, bidInfo)
			default:
				fmt.Printf("invalid collateral auction phase: %s", auction.GetPhase())
			}
		default:
			fmt.Printf("unsupported auction type: %s", auction.GetType())
		}
	}

	return auctionBidInfos
}

func handleForwardCollateralAuction(auction auctiontypes.Auction, keeper sdk.AccAddress, assetInfo map[string]AssetInfo, margin sdk.Dec) (AuctionInfo, bool) {
	collateralAuction := auction.(auctiontypes.CollateralAuction)

	// check bidder
	if collateralAuction.Bidder.Equals(keeper) {
		return AuctionInfo{}, false
	}

	assetInfoLot, ok := assetInfo[collateralAuction.Lot.Denom]
	if !ok {
		return AuctionInfo{}, false
	}

	assetInfoBid, ok := assetInfo[collateralAuction.MaxBid.Denom]
	if !ok {
		return AuctionInfo{}, false
	}

	proposedBid, ok := calculateProposedBid(collateralAuction.Lot, collateralAuction.MaxBid, assetInfoLot, assetInfoBid, margin)

	if !ok {
		return AuctionInfo{}, false
	}

	if proposedBid.IsZero() {
		return AuctionInfo{}, false
	}

	return AuctionInfo{
		ID:     collateralAuction.ID,
		Bidder: keeper,
		Amount: proposedBid,
	}, true
}

func handleReverseCollateralAuction(auction auctiontypes.Auction, keeper sdk.AccAddress, assetInfo map[string]AssetInfo, increment, margin sdk.Dec) (AuctionInfo, bool) {
	collateralAuction := auction.(auctiontypes.CollateralAuction)
	// check bidder
	if collateralAuction.Bidder.Equals(keeper) {
		return AuctionInfo{}, false
	}
	assetInfoLot, ok := assetInfo[collateralAuction.Lot.Denom]
	if !ok {
		fmt.Printf("lot asset info missing for %s, exiting\n", collateralAuction.Lot.Denom)
		return AuctionInfo{}, false
	}
	assetInfoBid, ok := assetInfo[collateralAuction.MaxBid.Denom]
	if !ok {
		fmt.Printf("bid asset info missing for %s, exiting\n", collateralAuction.MaxBid.Denom)
		return AuctionInfo{}, false
	}

	proposedLot, ok := calculateProposedLot(collateralAuction.Lot, collateralAuction.MaxBid, assetInfoLot, assetInfoBid, margin, increment)
	if !ok {
		return AuctionInfo{}, false
	}

	if proposedLot.IsZero() {
		return AuctionInfo{}, false
	}

	return AuctionInfo{
		ID:     collateralAuction.ID,
		Bidder: keeper,
		Amount: proposedLot,
	}, true
}

func calculateUSDValue(coin sdk.Coin, assetInfo AssetInfo) sdk.Dec {
	return coin.Amount.ToDec().Quo(assetInfo.ConversionFactor.ToDec()).Mul(assetInfo.Price)
}

func calculateProposedBid(lot, maxbid sdk.Coin, assetInfoLot, assetInfoBid AssetInfo, margin sdk.Dec) (sdk.Coin, bool) {
	bidsToTry := []sdk.Dec{d("1.0"), d("0.95"), d("0.9"), d("0.8"), d("0.7"), d("0.6"), d("0.5"), d("0.4"), d("0.3"), d("0.2"), d("0.1")}
	lotUSDValue := calculateUSDValue(lot, assetInfoLot)
	if lotUSDValue.IsZero() {
		return sdk.Coin{}, false
	}

	for _, bid := range bidsToTry {
		bidAmountInt := maxbid.Amount.ToDec().Mul(bid).TruncateInt()
		bidCoin := sdk.NewCoin(maxbid.Denom, bidAmountInt)
		bidUSDValue := calculateUSDValue(bidCoin, assetInfoBid)
		if sdk.OneDec().Sub((bidUSDValue.Quo(lotUSDValue))).GTE(margin) {
			return bidCoin, true
		}
	}
	return sdk.Coin{}, false
}

func calculateProposedLot(lot, maxbid sdk.Coin, assetInfoLot, assetInfoBid AssetInfo, margin, increment sdk.Dec) (sdk.Coin, bool) {
	bidUSDValue := calculateUSDValue(maxbid, assetInfoBid)
	if bidUSDValue.IsZero() {
		fmt.Printf("Exiting because of zero bid USD value\n")
		return sdk.Coin{}, false
	}
	incrementsToTry := []sdk.Dec{d("0.5"), d("0.4"), d("0.3"), d("0.2"), d("0.1"), d("0.05"), d("0.04"), d("0.03"), d("0.02"), increment}

	for _, lotIncrement := range incrementsToTry {

		proposedLotInt := lot.Amount.ToDec().Mul(sdk.OneDec().Sub(lotIncrement)).TruncateInt()
		proposedLotCoin := sdk.NewCoin(lot.Denom, proposedLotInt)
		proposedLotUSDValue := calculateUSDValue(proposedLotCoin, assetInfoLot)
		if proposedLotUSDValue.IsZero() {
			continue
		}
		if sdk.OneDec().Sub((bidUSDValue.Quo(proposedLotUSDValue))).GTE(margin) {
			fmt.Printf(`
				Increment tried: %s
				Proposed Lot: %s
				Proposed Lot USD Value: %s
				Bid USD Value: %s
		`, lotIncrement, proposedLotCoin, proposedLotUSDValue, bidUSDValue)
			return proposedLotCoin, true
		}
	}

	return sdk.Coin{}, false
}

func d(s string) sdk.Dec {
	return sdk.MustNewDecFromStr(s)
}
