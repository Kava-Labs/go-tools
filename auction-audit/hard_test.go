package main_test

import (
	"context"
	"testing"

	main "github.com/kava-labs/go-tools/auction-audit"
	"github.com/kava-labs/go-tools/auction-audit/config"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/require"
)

func TestGetAuctionSourceHardDeposit(t *testing.T) {
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

	sourceDeposit, height, err := main.GetAuctionSourceHARD(
		context.Background(),
		grpcClient,
		13807,
	)
	require.NoError(t, err)

	require.Equal(t, int64(9162), height)
	require.Equal(t, "kava1dpujcdhzfxykgzahuzzn9ywwdrlga5z2ggud6k", sourceDeposit.Depositor)
}
