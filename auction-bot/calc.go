package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"github.com/rs/zerolog"
)

const (
	USTDenom   = "ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C"
	atomDenom  = "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	aktDenom   = "ibc/799FDD409719A1122586A629AE8FCA17380351A51C1F47A80A1B8E7F2A491098"
	BusdDenom  = "busd"
	BtcbDenom  = "btcb"
	ukavaDenom = "ukava"
	hardDenom  = "hard"
	usdxDenom  = "usdx"
	swpDenom   = "swp"
)

type AuctionInfo struct {
	ID     uint64
	Bidder sdk.AccAddress
	Amount sdk.Coin
}

type AuctionInfos []AuctionInfo

func GetBids(
	logger zerolog.Logger,
	data *AuctionData,
	keeper sdk.AccAddress,
	margin sdk.Dec,
) AuctionInfos {
	var auctionBidInfos AuctionInfos
	var auctions int
	debt := sdk.NewCoin("debt", sdk.ZeroInt())
	for _, auction := range data.Auctions {
		// Only non-UST auctions
		if auction.GetLot().Denom == USTDenom {
			continue
		}
		auctions++
		da, ok := auction.(*auctiontypes.CollateralAuction)
		if ok {
			debt = debt.Add(da.CorrespondingDebt)
		}

		switch auction.GetType() {
		case auctiontypes.CollateralAuctionType:
			switch auction.GetPhase() {
			case auctiontypes.ForwardAuctionPhase:
				bidInfo, shouldBid := handleForwardCollateralAuction(
					auction,
					keeper,
					data.Assets,
					margin,
				)
				if !shouldBid {
					continue
				}
				auctionBidInfos = append(auctionBidInfos, bidInfo)
			case auctiontypes.ReverseAuctionPhase:
				bidInfo, shouldBid := handleReverseCollateralAuction(
					logger,
					auction,
					keeper,
					data.Assets,
					data.BidIncrement,
					margin,
				)
				if !shouldBid {
					continue
				}
				auctionBidInfos = append(auctionBidInfos, bidInfo)
			default:
				logger.Error().
					Str("phase", auction.GetPhase()).
					Msg("invalid collateral auction phase")
			}
		case auctiontypes.DebtAuctionType:
			bidInfo, shouldBid := handleReverseDebtAuction(
				logger,
				auction,
				keeper,
				data.Assets,
				data.BidIncrement,
				margin,
			)
			if !shouldBid {
				continue
			}
			auctionBidInfos = append(auctionBidInfos, bidInfo)
		default:
			logger.Error().
				Str("auction type", auction.GetType()).
				Msg("unsupported auction type")
		}
	}

	logger.Info().
		Int("auctions", auctions).
		Str("debt", debt.String()).
		Msg("checked auctions")

	return auctionBidInfos
}

func handleForwardCollateralAuction(
	auction auctiontypes.Auction,
	keeper sdk.AccAddress,
	assetInfo map[string]AssetInfo,
	margin sdk.Dec,
) (AuctionInfo, bool) {
	collateralAuction := auction.(*auctiontypes.CollateralAuction)
	assetInfoLot, ok := assetInfo[collateralAuction.Lot.Denom]
	if !ok {
		return AuctionInfo{}, false
	}

	assetInfoBid, ok := assetInfo[collateralAuction.MaxBid.Denom]
	if !ok {
		return AuctionInfo{}, false
	}

	proposedBid, ok := calculateProposedBid(
		collateralAuction.Bid,
		collateralAuction.Lot,
		collateralAuction.MaxBid,
		assetInfoLot,
		assetInfoBid,
		margin,
		collateralAuction.GetID(),
	)

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

func handleReverseCollateralAuction(
	logger zerolog.Logger,
	auction auctiontypes.Auction,
	keeper sdk.AccAddress,
	assetInfo map[string]AssetInfo,
	increment,
	margin sdk.Dec,
) (AuctionInfo, bool) {
	collateralAuction := auction.(*auctiontypes.CollateralAuction)
	assetInfoLot, ok := assetInfo[collateralAuction.Lot.Denom]
	if !ok {
		logger.Error().
			Uint64("auction id", auction.GetID()).
			Str("log denom", collateralAuction.Lot.Denom).
			Msg("lot asset info missing, exiting")

		return AuctionInfo{}, false
	}
	assetInfoBid, ok := assetInfo[collateralAuction.MaxBid.Denom]
	if !ok {
		logger.Error().
			Uint64("auction id", auction.GetID()).
			Str("maxbid denom", collateralAuction.MaxBid.Denom).
			Msg("max bid asset info missing, exiting")

		return AuctionInfo{}, false
	}

	proposedLot, ok := calculateProposedLot(
		logger,
		collateralAuction.Lot,
		collateralAuction.MaxBid,
		assetInfoLot,
		assetInfoBid,
		margin,
		increment,
		collateralAuction.GetID(),
	)
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

func handleReverseDebtAuction(
	logger zerolog.Logger,
	auction auctiontypes.Auction,
	keeper sdk.AccAddress,
	assetInfo map[string]AssetInfo,
	increment,
	margin sdk.Dec,
) (AuctionInfo, bool) {
	debtAuction := auction.(*auctiontypes.DebtAuction)
	assetInfoLot, ok := assetInfo[debtAuction.Lot.Denom]
	if !ok {
		logger.Error().
			Uint64("auction id", debtAuction.ID).
			Str("log denom", debtAuction.Lot.Denom).
			Msg("reverse debt lot asset info missing")

		return AuctionInfo{}, false
	}
	assetInfoBid, ok := assetInfo[debtAuction.Bid.Denom]
	if !ok {
		logger.Error().
			Uint64("auction id", debtAuction.ID).
			Str("bid denom", debtAuction.Bid.Denom).
			Msg("reverse debt lot bid asset info missing")

		return AuctionInfo{}, false
	}

	proposedLot, ok := calculateProposedLot(
		logger,
		debtAuction.Lot,
		debtAuction.Bid,
		assetInfoLot,
		assetInfoBid,
		margin,
		increment,
		debtAuction.GetID(),
	)
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
	return coin.Amount.ToLegacyDec().Quo(assetInfo.ConversionFactor.ToLegacyDec()).Mul(assetInfo.Price)
}

func calculateProposedBid(
	currentBid, lot, maxbid sdk.Coin,
	assetInfoLot, assetInfoBid AssetInfo,
	margin sdk.Dec,
	id uint64,
) (sdk.Coin, bool) {
	bidsToTry := []sdk.Dec{d("1.0"), d("0.95"), d("0.9"), d("0.8"), d("0.7"), d("0.6"), d("0.5"), d("0.4"), d("0.3"), d("0.2"), d("0.1")}
	lotUSDValue := calculateUSDValue(lot, assetInfoLot)
	if lotUSDValue.IsZero() {
		return sdk.Coin{}, false
	}
	minBid := currentBid.Amount.ToLegacyDec().Mul(d("1.0105")).RoundInt()
	if minBid.GT(maxbid.Amount) {
		minBid = maxbid.Amount
	}

	for _, bidIncrement := range bidsToTry {
		bidAmountInt := maxbid.Amount.ToLegacyDec().Mul(bidIncrement).TruncateInt()
		if bidAmountInt.LT(minBid) {
			bidAmountInt = minBid
		}
		bidCoin := sdk.NewCoin(maxbid.Denom, bidAmountInt)
		bidUSDValue := calculateUSDValue(bidCoin, assetInfoBid)
		if sdk.OneDec().Sub((bidUSDValue.Quo(lotUSDValue))).GTE(margin) {
			// 			fmt.Printf(`
			// 	Auction id: %d
			// 	Increment tried: %s
			// 	Proposed Bid: %s
			// 	Proposed Bid USD Value: %s
			// 	Lot USD Value: %s
			// `,
			// 				id, bidIncrement, bidCoin, bidUSDValue, lotUSDValue,
			// 			)
			return bidCoin, true
		}
	}
	return sdk.Coin{}, false
}

func calculateProposedLot(
	logger zerolog.Logger,
	lot, maxbid sdk.Coin,
	assetInfoLot, assetInfoBid AssetInfo,
	margin, increment sdk.Dec,
	id uint64,
) (sdk.Coin, bool) {
	bidUSDValue := calculateUSDValue(maxbid, assetInfoBid)
	if bidUSDValue.IsZero() {
		logger.Info().
			Uint64("auction id", id).
			Msg("Exiting auction because of zero bid USD value")

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
		proposedLotInt := lot.Amount.ToLegacyDec().Mul(sdk.OneDec().Sub(lotIncrement)).TruncateInt()
		proposedLotCoin := sdk.NewCoin(lot.Denom, proposedLotInt)
		proposedLotUSDValue := calculateUSDValue(proposedLotCoin, assetInfoLot)
		if proposedLotUSDValue.IsZero() {
			continue
		}
		if sdk.OneDec().Sub((bidUSDValue.Quo(proposedLotUSDValue))).GTE(margin) {
			// 			fmt.Printf(`
			// 	Auction id: %d
			// 	Increment tried: %s
			// 	Proposed Lot: %s
			// 	Proposed Lot USD Value: %s
			// 	Bid USD Value: %s
			// `,
			// 				id, lotIncrement, proposedLotCoin, proposedLotUSDValue, bidUSDValue,
			// 			)
			return proposedLotCoin, true
		}
	}

	return sdk.Coin{}, false
}

func d(s string) sdk.Dec {
	return sdk.MustNewDecFromStr(s)
}
