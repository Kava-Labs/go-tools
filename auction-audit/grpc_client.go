package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"

	sdkClient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	tmclient "github.com/tendermint/tendermint/rpc/client"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type GrpcClient struct {
	cdc            codec.Codec
	Decoder        sdkClient.TxConfig
	GrpcClientConn *grpc.ClientConn
	Tm             tmservice.ServiceClient
	Auction        auctiontypes.QueryClient
	Tx             txtypes.ServiceClient
	Pricefeed      pricefeedtypes.QueryClient

	// rpc client for tendermint rpc
	Tendermint tmclient.SignClient
}

func connectGrpc(grpcTarget string) (*grpc.ClientConn, error) {
	grpcUrl, err := url.Parse(grpcTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grpc url: %s", err)
	}

	var secureOpt grpc.DialOption
	switch grpcUrl.Scheme {
	case "http":
		secureOpt = grpc.WithInsecure()
	case "https":
		creds := credentials.NewTLS(&tls.Config{})
		secureOpt = grpc.WithTransportCredentials(creds)
	default:
		log.Fatalf("unknown rpc url scheme %s\n", grpcUrl.Scheme)
	}

	grpcConn, err := grpc.Dial(grpcUrl.Host, secureOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to dial grpc server: %s", err)
	}

	return grpcConn, nil
}

func NewGrpcClient(
	grpcTarget string,
	rpcTarget string,
	cdc codec.Codec,
	txConfig sdkClient.TxConfig,
) (GrpcClient, error) {
	grpcConn, err := connectGrpc(grpcTarget)
	if err != nil {
		return GrpcClient{}, err
	}

	rpcClient, err := rpchttpclient.New(rpcTarget, "/websocket")
	if err != nil {
		return GrpcClient{}, err
	}

	return GrpcClient{
		cdc:            cdc,
		Decoder:        txConfig,
		GrpcClientConn: grpcConn,
		Tm:             tmservice.NewServiceClient(grpcConn),
		Auction:        auctiontypes.NewQueryClient(grpcConn),
		Tx:             txtypes.NewServiceClient(grpcConn),
		Pricefeed:      pricefeedtypes.NewQueryClient(grpcConn),

		Tendermint: rpcClient,
	}, nil
}

func (c GrpcClient) GetBeginBlockEventsFromQuery(
	ctx context.Context,
	query string,
) (sdk.StringEvents, int64, error) {
	// 1) Block search to find auction_start event and corresponding height
	// https://rpc.kava.io/block_search?query=%22auction_start.auction_id=16837%22

	blocks, err := c.QueryBlock(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	if len(blocks) == 0 {
		return nil, 0, fmt.Errorf("no blocks found")
	}

	// 2) Block results to query events from height
	// https://rpc.kava.io/block_results?height=3146803
	events, err := c.GetBeginBlockEvents(ctx, blocks[0].Block.Height)
	return events, blocks[0].Block.Height, err
}

func (c GrpcClient) QueryBlock(ctx context.Context, query string) ([]*coretypes.ResultBlock, error) {
	page := 1
	perPage := 100

	res, err := c.Tendermint.BlockSearch(
		ctx,
		query,
		&page,
		&perPage,
		"desc",
	)

	if err != nil {
		return nil, fmt.Errorf("failed BlockSearch: %w", err)
	}

	return res.Blocks, nil
}

func (c GrpcClient) GetBeginBlockEvents(ctx context.Context, height int64) (sdk.StringEvents, error) {
	res, err := c.Tendermint.BlockResults(
		ctx,
		&height,
	)

	if err != nil {
		return nil, fmt.Errorf("failed BlockResults: %w", err)
	}

	// Do not use sdk.StringifyEvents as it flattens events which makes it
	// more difficult to parse.
	strEvents := make(sdk.StringEvents, 0, len(res.BeginBlockEvents))
	for _, e := range res.BeginBlockEvents {
		strEvents = append(strEvents, sdk.StringifyEvent(e))
	}

	return strEvents, nil
}
