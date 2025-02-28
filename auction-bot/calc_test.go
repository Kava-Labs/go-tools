package main

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// c is a helper function to create sdk.Coin
var c = sdk.NewInt64Coin
var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

func TestMain(m *testing.M) {
	// Set kava bech32 prefix
	app.SetSDKConfig()

	os.Exit(m.Run())
}

func TestGetBids(t *testing.T) {
	testCases := []struct {
		name         string
		auctionData  AuctionData
		keeper       sdk.AccAddress
		margin       sdk.Dec
		expectedBids AuctionInfos
	}{
		{
			name: "single forward auction",
			auctionData: AuctionData{
				Auctions: []auctiontypes.Auction{
					&auctiontypes.CollateralAuction{
						BaseAuction: auctiontypes.BaseAuction{
							ID:  0,
							Lot: c("bnb", 1000e8),
							Bid: c("usdx", 0),
						},
						MaxBid:            c("usdx", 220_000e6),
						CorrespondingDebt: c("debt", 200_000e6),
					},
				},
				Assets: map[string]AssetInfo{
					"usdx": {
						Price:            d("1.00"),
						ConversionFactor: sdk.NewInt(1e6),
					},
					"bnb": {
						Price:            d("200"),
						ConversionFactor: sdk.NewInt(1e8),
					},
				},
				BidIncrement: d("0.01"),
			},
			margin: d("0.05"),
			expectedBids: AuctionInfos{{
				ID:     0,
				Amount: c("usdx", 176_000e6),
			}},
		},
		{
			name: "min increment forward auction",
			auctionData: AuctionData{
				Auctions: []auctiontypes.Auction{
					&auctiontypes.CollateralAuction{
						BaseAuction: auctiontypes.BaseAuction{
							ID:  0,
							Lot: c("xrp", 25),
							Bid: c("ukava", 1),
						},
						MaxBid:            c("ukava", 4),
						CorrespondingDebt: c("debt", 0),
					},
				},
				Assets: map[string]AssetInfo{
					"xrp": {
						Price:            d("2.00"),
						ConversionFactor: sdk.NewInt(1e8),
					},
					"ukava": {
						Price:            d("0.44"),
						ConversionFactor: sdk.NewInt(1e6),
					},
				},
				BidIncrement: d("0.01"),
			},
			margin: d("0.05"),
			expectedBids: AuctionInfos{{
				ID:     0,
				Amount: c("ukava", 2),
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testAddr, err := sdk.AccAddressFromBech32("kava10eup8kvq26z8ekjj9rkplr2lwwynskftqc4ytv")
			require.NoError(t, err)

			actualBids := GetBids(logger, &tc.auctionData, testAddr, tc.margin)

			// add in expected bidder address here to keep test cases simple
			for i := 0; i < len(tc.expectedBids); i++ {
				tc.expectedBids[i].Bidder = testAddr
			}
			require.Equal(t, tc.expectedBids, actualBids)
		})
	}
}

func TestCalculateProposedBid(t *testing.T) {
	assetInfos := map[string]AssetInfo{
		"usdx": {
			Price:            d("1.00"),
			ConversionFactor: sdk.NewInt(1e6),
		},
		"bnb": {
			Price:            d("200.00"),
			ConversionFactor: sdk.NewInt(1e8),
		},
	}

	testCases := []struct {
		name string

		lot, bid, maxBid sdk.Coin
		margin           sdk.Dec

		expectedOk  bool
		expectedBid sdk.Coin
	}{
		{
			name:        "normal forward",
			lot:         c("bnb", 1000e8),
			bid:         c("usdx", 0),
			maxBid:      c("usdx", 220_000e6),
			margin:      d("0.05"),
			expectedOk:  true,
			expectedBid: c("usdx", 176_000e6),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var testID uint64

			actualBid, ok := calculateProposedBid(
				tc.bid,
				tc.lot,
				tc.maxBid,
				assetInfos[tc.lot.Denom],
				assetInfos[tc.bid.Denom],
				tc.margin,
				testID,
			)
			require.Equal(t, tc.expectedOk, ok)
			if ok {
				// only care about returned bid if ok
				require.Equal(t, tc.expectedBid, actualBid)
			}
		})
	}
}

func TestCalculateProposedLot(t *testing.T) {
	assetInfos := map[string]AssetInfo{
		"usdx": {
			Price:            d("1.00"),
			ConversionFactor: sdk.NewInt(1e6),
		},
		"bnb": {
			Price:            d("200.00"),
			ConversionFactor: sdk.NewInt(1e8),
		},
	}

	testCases := []struct {
		name string

		lot, maxBid sdk.Coin
		margin      sdk.Dec

		expectedOk  bool
		expectedLot sdk.Coin
	}{
		{
			name:        "normal reverse",
			lot:         c("bnb", 1000e8),
			maxBid:      c("usdx", 100_000e6),
			margin:      d("0.05"),
			expectedOk:  true,
			expectedLot: c("bnb", 600e8),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var testID uint64

			actualLot, ok := calculateProposedLot(
				logger,
				tc.lot,
				tc.maxBid,
				assetInfos[tc.lot.Denom],
				assetInfos[tc.maxBid.Denom],
				tc.margin,
				d("0.01"),
				testID,
			)
			require.Equal(t, tc.expectedOk, ok)
			if ok {
				// only care about returned lot if ok
				require.Equal(t, tc.expectedLot, actualLot)
			}
		})
	}
}
