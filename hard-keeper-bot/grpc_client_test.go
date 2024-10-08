package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var latestHeight int64
var grpcClient *GrpcClient

func TestMain(m *testing.M) {
	grpcClient, _ = NewGrpcClient("https://grpc.kava.io:443")

	os.Exit(m.Run())
}

func TestHardKeeperGetInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	res, err := grpcClient.GetInfo()
	require.NoError(t, err)
	require.Greater(t, res.LatestHeight, int64(11000000))
	require.Equal(t, "kava_2222-10", res.ChainId)
	latestHeight = res.LatestHeight
}

func TestHardKeeperGetPrices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	res, err := grpcClient.GetPrices(latestHeight)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	require.Equal(t, len(res), 29)
}

func TestHardKeeperGetMarkets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	res, err := grpcClient.GetMarkets(latestHeight)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	require.Equal(t, len(res), 16)
}

func TestHardKeeperGetBorrows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	res, err := grpcClient.GetBorrows(latestHeight)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	require.Equal(t, len(res), 100)
}

func TestHardKeeperGetDeposits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	res, err := grpcClient.GetDeposits(latestHeight)
	require.NoError(t, err)
	require.NotEmpty(t, res)
	require.Equal(t, len(res), 100)
}
