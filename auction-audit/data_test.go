package main_test

import (
	"context"
	"testing"

	main "github.com/kava-labs/go-tools/auction-audit"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/require"
)

func TestGetAuctionSourceCDP(t *testing.T) {
	app.SetSDKConfig()
	config, err := main.LoadConfig(&main.EnvLoader{})
	require.NoError(t, err)

	encodingConfig := app.MakeEncodingConfig()

	grpcClient := main.NewGrpcClient(
		config.GrpcURL,
		encodingConfig.Marshaler,
		encodingConfig.TxConfig,
	)

	sourceCDP, err := main.GetAuctionSourceCDP(
		context.Background(),
		grpcClient,
		2824780,
		16596,
	)
	require.NoError(t, err)

	t.Logf("source cdp %v", sourceCDP)

	require.Equal(t, uint64(13188), sourceCDP.ID)
}
