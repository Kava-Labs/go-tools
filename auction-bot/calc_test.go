package main

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"github.com/stretchr/testify/require"
)

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
							Lot: sdk.NewInt64Coin("bnb", 1000e8),
							Bid: sdk.NewInt64Coin("usdx", 0),
						},
						MaxBid:            sdk.NewInt64Coin("usdx", 220_000e6),
						CorrespondingDebt: sdk.NewInt64Coin("debt", 200_000e6),
					},
				},
				Assets: map[string]AssetInfo{
					"usdx": {
						Price:            sdk.MustNewDecFromStr("1.00"),
						ConversionFactor: sdk.NewInt(1e6),
					},
					"bnb": {
						Price:            sdk.MustNewDecFromStr("200"),
						ConversionFactor: sdk.NewInt(1e8),
					},
				},
				BidIncrement: sdk.MustNewDecFromStr("0.01"),
			},
			margin: sdk.MustNewDecFromStr("0.05"),
			expectedBids: AuctionInfos{{
				ID:     0,
				Amount: sdk.NewInt64Coin("usdx", 176_000e6),
			}},
		},
	}

	for _, tc := range testCases {
		testAddr, err := sdk.AccAddressFromBech32("kava10eup8kvq26z8ekjj9rkplr2lwwynskftqc4ytv")
		require.NoError(t, err)

		actualBids := GetBids(&tc.auctionData, testAddr, tc.margin)

		// add in expected bidder address here to keep test cases simple
		for i := 0; i < len(tc.expectedBids); i++ {
			tc.expectedBids[i].Bidder = testAddr
		}
		require.Equal(t, tc.expectedBids, actualBids)
	}
}
