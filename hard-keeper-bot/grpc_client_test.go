package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var invalidHeight int64 = 100
var latestHeight int64
var grpcClient *GrpcClient

func TestMain(m *testing.M) {
	grpcClient, _ = NewGrpcClient("https://grpc.kava.io:443")

	os.Exit(m.Run())
}

func TestGrpcClientInvalidUrl(t *testing.T) {
	_, err := NewGrpcClient("invalid-url")
	require.Error(t, err)
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

func TestHardKeeperGetPricesInvalidHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	_, err := grpcClient.GetPrices(invalidHeight)
	require.Error(t, err)
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

func TestHardKeeperGetMarketsInvalidHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	_, err := grpcClient.GetMarkets(invalidHeight)
	require.Error(t, err)
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

func TestHardKeeperGetBorrowsInvalidHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	_, err := grpcClient.GetBorrows(invalidHeight)
	require.Error(t, err)
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

func TestHardKeeperGetDepositsInvalidHeight(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	_, err := grpcClient.GetDeposits(invalidHeight)
	require.Error(t, err)
}
