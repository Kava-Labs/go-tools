package main_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	main "github.com/kava-labs/go-tools/auction-audit"
	"github.com/kava-labs/go-tools/auction-audit/config"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAuctionSourceCDP(t *testing.T) {
	app.SetSDKConfig()
	config, err := config.LoadConfig(&config.EnvLoader{})
	require.NoError(t, err)

	encodingConfig := app.MakeEncodingConfig()

	grpcClient, err := main.NewGrpcClient(
		config.GrpcURL,
		config.RpcURL,
		encodingConfig.Marshaler,
		encodingConfig.TxConfig,
	)
	require.NoError(t, err)

	tests := []struct {
		name             string
		giveAuctionID    uint64
		wantSourceHeight int64
		wantCdpID        uint64
	}{
		{
			name:             "CDP auction via MsgLiquidate",
			giveAuctionID:    16596,
			wantSourceHeight: 2824779,
			wantCdpID:        13188,
		},
		{
			// Auction that was started in cdp BeginBlocker which cannot be
			// queried for source cdp via grpc txs
			name:             "CDP auction via BeginBlocker",
			giveAuctionID:    16837,
			wantSourceHeight: 3146802, // 1 block before auction was started
			wantCdpID:        23216,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceCDP, height, err := main.GetAuctionSourceCDP(
				context.Background(),
				grpcClient,
				tt.giveAuctionID,
			)
			require.NoError(t, err)
			t.Logf("source cdp %v", sourceCDP.Collateral)

			assert.Equal(t, tt.wantSourceHeight, height)
			assert.Equal(t, tt.wantCdpID, sourceCDP.ID)
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
