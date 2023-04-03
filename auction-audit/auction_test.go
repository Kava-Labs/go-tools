package main_test

import (
	"context"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	main "github.com/kava-labs/go-tools/auction-audit"
	"github.com/kava-labs/go-tools/auction-audit/config"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestGetAuctionEndDataOpt(t *testing.T) {
	app.SetSDKConfig()
	config, err := config.LoadConfig(&config.EnvLoader{})
	require.NoError(t, err)

	encodingConfig := app.MakeEncodingConfig()

	client, err := main.NewClient(
		config.RpcURL,
		encodingConfig.Amino,
	)
	require.NoError(t, err)

	endData, err := main.GetAuctionEndData(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), client, 2772000, 2787000)
	require.NoError(t, err)

	t.Logf("endData: %v", endData)
	t.Logf("len(endData): %v", len(endData))
}

func TestGetAuctionSourceCDP(t *testing.T) {
	app.SetSDKConfig()
	config, err := config.LoadConfig(&config.EnvLoader{})
	require.NoError(t, err)

	encodingConfig := app.MakeEncodingConfig()

	client, err := main.NewClient(
		config.RpcURL,
		encodingConfig.Amino,
	)
	require.NoError(t, err)

	tests := []struct {
		name             string
		giveAuctionID    uint64
		wantSourceHeight int64
		wantAmount       sdk.Coin
	}{
		{
			name:             "CDP auction via MsgLiquidate",
			giveAuctionID:    16596,
			wantSourceHeight: 2824780,
			wantAmount:       sdk.NewCoin("busd", sdk.NewInt(1099890000)),
		},
		{
			// Auction that was started in cdp BeginBlocker which cannot be
			// queried for source cdp via grpc txs
			name:             "CDP auction via BeginBlocker",
			giveAuctionID:    16837,
			wantSourceHeight: 3146803,
			wantAmount:       sdk.NewCoin("xrpb", sdk.NewInt(475595214356)),
		},
		{
			name:             "CDP auction 2 via BeginBlocker",
			giveAuctionID:    16017,
			wantSourceHeight: 2773444,
			wantAmount:       sdk.NewCoin("xrpb", sdk.NewInt(246056832094)),
		},
		{
			name:             "CDP auction 2 via BeginBlocker",
			giveAuctionID:    15265,
			wantSourceHeight: 2773444,
			wantAmount:       sdk.NewCoin("xrpb", sdk.NewInt(246056832094)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, height, err := main.GetAuctionStartLotCDP(
				context.Background(),
				client,
				tt.giveAuctionID,
			)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSourceHeight, height)
			assert.Equal(t, tt.wantAmount, amount)
		})
	}

}

func TestGetOriginalAmountPercentSub(t *testing.T) {
	tests := []struct {
		givePercent        sdk.Dec
		giveAmount         sdk.Dec
		wantOriginalAmount sdk.Int
	}{
		{
			givePercent:        sdk.NewDecWithPrec(1, 2),
			giveAmount:         sdk.NewDec(5453056340),
			wantOriginalAmount: sdk.NewInt(5508137717),
		},
		{
			givePercent:        sdk.NewDecWithPrec(1, 2),
			giveAmount:         sdk.NewDec(999901600),
			wantOriginalAmount: sdk.NewInt(1010001616),
		},
		{
			givePercent:        sdk.NewDecWithPrec(1, 2),
			giveAmount:         sdk.NewDec(1099890000),
			wantOriginalAmount: sdk.NewInt(1111000000),
		},
	}

	for _, tt := range tests {
		orig := main.GetOriginalAmountPercentSub(
			tt.givePercent,
			tt.giveAmount,
		)

		require.Equal(t, tt.wantOriginalAmount, orig.RoundInt())
	}

}
