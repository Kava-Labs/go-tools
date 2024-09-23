package auctions

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/kava-labs/kava/app"
	kava "github.com/kava-labs/kava/app"
	kavagrpc "github.com/kava-labs/kava/client/grpc"

	"github.com/stretchr/testify/require"
)

// pruning node stores only latest data and doesn't store historical data.
// It is much more performant and can be used for transaction search, as on archive node it is very slow.
var pruningQueryClient AuctionClient

// we should use dataQueryClient only for historical data, as pruning node doesn't contain historical data
var dataQueryClient AuctionClient

func TestMain(m *testing.M) {
	app.SetSDKConfig()

	// Create codec for messages
	encodingConfig := kava.MakeEncodingConfig()

	pruningGRPCClient, err := kavagrpc.NewClient("https://grpc.kava.io:443")
	if err != nil {
		log.Fatalf("grpc client failed to connect %s", err)
	}

	dataGRPCClient, err := kavagrpc.NewClient("https://grpc.data.infra.kava.io:443")
	if err != nil {
		log.Fatalf("grpc client failed to connect %s", err)
	}

	pruningQueryClient = NewGrpcAuctionClient(pruningGRPCClient, encodingConfig.Marshaler)
	dataQueryClient = NewGrpcAuctionClient(dataGRPCClient, encodingConfig.Marshaler)

	os.Exit(m.Run())
}

func TestGrpcGetInfo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	info, err := pruningQueryClient.GetInfo()
	require.NoError(t, err)
	require.Greater(t, info.LatestHeight, int64(11000000))
	require.Equal(t, "kava_2222-10", info.ChainId)
}

func TestGrpcGetPrices(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	prices, err := dataQueryClient.GetPrices(11000000)
	require.NoError(t, err)
	require.Len(t, prices, 29)
	for _, price := range prices {
		require.NotEmpty(t, price.MarketID)
		require.NotEmpty(t, price.Price)
	}
}

func TestGrpcGetAuctions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	auctions, err := pruningQueryClient.GetAuctions(11000000)
	require.NoError(t, err)
	require.Len(t, auctions, 10)
	for _, auction := range auctions {
		require.NotEmpty(t, auction.GetID())
		require.NotEmpty(t, auction.GetInitiator())
		require.NotEmpty(t, auction.GetLot())
		require.NotEmpty(t, auction.GetBid())
		require.NotEmpty(t, auction.GetEndTime())
	}
}

func TestGrpcGetMarkets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	markets, err := dataQueryClient.GetMarkets(11000000)
	require.NoError(t, err)
	jsonMarkets, err := json.Marshal(markets)
	fmt.Println(len(markets))
	fmt.Println(string(jsonMarkets))
	require.Len(t, markets, 10)
}

func TestGrpcGetMoneyMarkets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	moneyMarkets, err := dataQueryClient.GetMoneyMarkets(11000000)
	require.NoError(t, err)
	require.Len(t, moneyMarkets, 29)
}
