package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/url"

	sdkClient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	tmclient "github.com/tendermint/tendermint/rpc/client"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
)

type GrpcClient struct {
	cdc            codec.Codec
	Decoder        sdkClient.TxConfig
	GrpcClientConn *grpc.ClientConn
	Tm             tmservice.ServiceClient
	Auction        auctiontypes.QueryClient
	Tx             txtypes.ServiceClient
	CDP            cdptypes.QueryClient
	Hard           hardtypes.QueryClient
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
		CDP:            cdptypes.NewQueryClient(grpcConn),
		Hard:           hardtypes.NewQueryClient(grpcConn),
		Pricefeed:      pricefeedtypes.NewQueryClient(grpcConn),

		Tendermint: rpcClient,
	}, nil
}
