package main_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	main "github.com/kava-labs/go-tools/auction-audit"
	"github.com/kava-labs/go-tools/auction-audit/config"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/require"
)

func TestGetAuctionSourceCDP(t *testing.T) {
	app.SetSDKConfig()
	config, err := config.LoadConfig(&config.EnvLoader{})
	require.NoError(t, err)

	encodingConfig := app.MakeEncodingConfig()

	grpcClient := main.NewGrpcClient(
		config.GrpcURL,
		encodingConfig.Marshaler,
		encodingConfig.TxConfig,
	)

	sourceCDP, height, err := main.GetAuctionSourceCDP(
		context.Background(),
		grpcClient,
		16596,
	)
	require.NoError(t, err)
	t.Logf("source cdp %v", sourceCDP.Collateral)

	require.Equal(t, uint64(2824779), height)
	require.Equal(t, uint64(13188), sourceCDP.ID)
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
