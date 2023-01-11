package main_test

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	main "github.com/kava-labs/go-tools/auction-audit"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/require"
)

func TestGetTotalCoinsUsdValueAtHeight(t *testing.T) {
	app.SetSDKConfig()
	config, err := main.LoadConfig(&main.EnvLoader{})
	require.NoError(t, err)

	encodingConfig := app.MakeEncodingConfig()

	grpcClient := main.NewGrpcClient(
		config.GrpcURL,
		encodingConfig.Marshaler,
		encodingConfig.TxConfig,
	)

	tests := []struct {
		giveCoins         sdk.Coins
		giveHeight        int64
		wantTotalUsdValue sdk.Dec
	}{
		{
			giveCoins: sdk.NewCoins(
				sdk.NewCoin("ukava", sdk.NewInt(100)),
			),
			giveHeight:        3126375,
			wantTotalUsdValue: sdk.MustNewDecFromStr("74.6499999999999996"),
		},
		{
			giveCoins: sdk.NewCoins(
				sdk.NewCoin("ukava", sdk.NewInt(100)),
				sdk.NewCoin("bnb", sdk.NewInt(50)),
			),
			giveHeight: 3120000,
			// Add up prices for both
			wantTotalUsdValue: sdk.MustNewDecFromStr("74.1499999999999992").
				Add(sdk.MustNewDecFromStr("275.099999999999994316").
					MulInt64(50),
				),
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s at height %d", tt.giveCoins, tt.giveHeight), func(t *testing.T) {
			usdValue, err := main.GetTotalCoinsUsdValueAtHeight(
				grpcClient,
				tt.giveHeight,
				tt.giveCoins,
				main.Spot,
			)
			require.NoError(t, err)

			require.Equal(t, tt.wantTotalUsdValue, usdValue)
		})
	}
}
