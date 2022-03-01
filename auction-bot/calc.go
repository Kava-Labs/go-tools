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
				fmt.Printf("invalid collateral auction phase: %s\n", auction.GetPhase())
			}
		case auctiontypes.DebtAuctionType:
			bidInfo, shouldBid := handleReverseDebtAuction(auction, keeper, data.Assets, data.BidIncrement, margin)
			if !shouldBid {
				continue
			}
			auctionBidInfos = append(auctionBidInfos, bidInfo)
		default:
			fmt.Printf("unsupported auction type: %s\n", auction.GetType())
		}
	}

	return auctionBidInfos
}

func handleForwardCollateralAuction(auction auctiontypes.Auction, keeper sdk.AccAddress, assetInfo map[string]AssetInfo, margin sdk.Dec) (AuctionInfo, bool) {
	collateralAuction := auction.(*auctiontypes.CollateralAuction)
	assetInfoLot, ok := assetInfo[collateralAuction.Lot.Denom]
	if !ok {
		return AuctionInfo{}, false
	}

	assetInfoBid, ok := assetInfo[collateralAuction.MaxBid.Denom]
	if !ok {
		return AuctionInfo{}, false
	}

	proposedBid, ok := calculateProposedBid(collateralAuction.Bid, collateralAuction.Lot, collateralAuction.MaxBid, assetInfoLot, assetInfoBid, margin, collateralAuction.GetID())

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
	collateralAuction := auction.(*auctiontypes.CollateralAuction)
	assetInfoLot, ok := assetInfo[collateralAuction.Lot.Denom]
	if !ok {
		fmt.Printf("lot asset info missing, exiting")
		return AuctionInfo{}, false
	}
	assetInfoBid, ok := assetInfo[collateralAuction.MaxBid.Denom]
	if !ok {
		fmt.Printf("bid asset info missing, exiting")
		return AuctionInfo{}, false
	}

	proposedLot, ok := calculateProposedLot(collateralAuction.Lot, collateralAuction.MaxBid, assetInfoLot, assetInfoBid, margin, increment, collateralAuction.GetID())
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

func handleReverseDebtAuction(auction auctiontypes.Auction, keeper sdk.AccAddress, assetInfo map[string]AssetInfo, increment, margin sdk.Dec) (AuctionInfo, bool) {
	debtAuction := auction.(*auctiontypes.DebtAuction)
	assetInfoLot, ok := assetInfo[debtAuction.Lot.Denom]
	if !ok {
		fmt.Printf("lot asset info missing, exiting")
		return AuctionInfo{}, false
	}
	assetInfoBid, ok := assetInfo[debtAuction.Bid.Denom]
	if !ok {
		fmt.Printf("bid asset info missing, exiting")
		return AuctionInfo{}, false
	}

	proposedLot, ok := calculateProposedLot(debtAuction.Lot, debtAuction.Bid, assetInfoLot, assetInfoBid, margin, increment, debtAuction.GetID())
	if !ok {
		return AuctionInfo{}, false
	}

	if proposedLot.IsZero() {
		return AuctionInfo{}, false
	}

	return AuctionInfo{
		ID:     debtAuction.ID,
		Bidder: keeper,
		Amount: proposedLot,
	}, true
}

func calculateUSDValue(coin sdk.Coin, assetInfo AssetInfo) sdk.Dec {
	return coin.Amount.ToDec().Quo(assetInfo.ConversionFactor.ToDec()).Mul(assetInfo.Price)
}

func calculateProposedBid(currentBid, lot, maxbid sdk.Coin, assetInfoLot, assetInfoBid AssetInfo, margin sdk.Dec, id uint64) (sdk.Coin, bool) {
	bidsToTry := []sdk.Dec{d("1.0"), d("0.95"), d("0.9"), d("0.8"), d("0.7"), d("0.6"), d("0.5"), d("0.4"), d("0.3"), d("0.2"), d("0.1")}
	lotUSDValue := calculateUSDValue(lot, assetInfoLot)
	if lotUSDValue.IsZero() {
		return sdk.Coin{}, false
	}
	minBid := currentBid.Amount.ToDec().Mul(d("1.0105")).RoundInt()
	if minBid.GT(maxbid.Amount) {
		minBid = maxbid.Amount
	}

	for _, bidIncrement := range bidsToTry {
		bidAmountInt := maxbid.Amount.ToDec().Mul(bidIncrement).TruncateInt()
		if bidAmountInt.LT(minBid) {
			bidAmountInt = minBid
		}
		bidCoin := sdk.NewCoin(maxbid.Denom, bidAmountInt)
		bidUSDValue := calculateUSDValue(bidCoin, assetInfoBid)
		if sdk.OneDec().Sub((bidUSDValue.Quo(lotUSDValue))).GTE(margin) {
			fmt.Printf(`
	Auction id: %d
	Increment tried: %s
	Proposed Bid: %s
	Proposed Bid USD Value: %s
	Lot USD Value: %s
`,
				id, bidIncrement, bidCoin, bidUSDValue, lotUSDValue,
			)
			return bidCoin, true
		}
	}
	return sdk.Coin{}, false
}

func calculateProposedLot(lot, maxbid sdk.Coin, assetInfoLot, assetInfoBid AssetInfo, margin, increment sdk.Dec, id uint64) (sdk.Coin, bool) {
	bidUSDValue := calculateUSDValue(maxbid, assetInfoBid)
	if bidUSDValue.IsZero() {
		fmt.Printf("Exiting auction %d because of zero bid USD value\n", id)
		return sdk.Coin{}, false
	}
	incrementsToTry := []sdk.Dec{
		d("0.5"),
		d("0.4"),
		d("0.3"),
		d("0.2"),
		d("0.1"),
		d("0.05"),
		d("0.04"),
		d("0.03"),
		d("0.02"),
		increment,
	}

	for _, lotIncrement := range incrementsToTry {
		proposedLotInt := lot.Amount.ToDec().Mul(sdk.OneDec().Sub(lotIncrement)).TruncateInt()
		proposedLotCoin := sdk.NewCoin(lot.Denom, proposedLotInt)
		proposedLotUSDValue := calculateUSDValue(proposedLotCoin, assetInfoLot)
		if proposedLotUSDValue.IsZero() {
			continue
		}
		if sdk.OneDec().Sub((bidUSDValue.Quo(proposedLotUSDValue))).GTE(margin) {
			fmt.Printf(`
	Auction id: %d
	Increment tried: %s
	Proposed Lot: %s
	Proposed Lot USD Value: %s
	Bid USD Value: %s
`,
				id, lotIncrement, proposedLotCoin, proposedLotUSDValue, bidUSDValue,
			)
			return proposedLotCoin, true
		}
	}

	return sdk.Coin{}, false
}

func d(s string) sdk.Dec {
	return sdk.MustNewDecFromStr(s)
}
