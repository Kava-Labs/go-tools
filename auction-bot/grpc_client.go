package main

import (
	"crypto/tls"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GrpcClient struct {
	Tm        tmservice.ServiceClient
	Auction   auctiontypes.QueryClient
	Cdp       cdptypes.QueryClient
	Hard      hardtypes.QueryClient
	Pricefeed pricefeedtypes.QueryClient
}

func NewGrpcClient(target string) GrpcClient {
	creds := credentials.NewTLS(&tls.Config{})
	grpcConn, err := grpc.Dial(target, grpc.WithTransportCredentials(creds))
	if err != nil {
		panic(err)
	}

	return GrpcClient{
		Tm:        tmservice.NewServiceClient(grpcConn),
		Auction:   auctiontypes.NewQueryClient(grpcConn),
		Cdp:       cdptypes.NewQueryClient(grpcConn),
		Hard:      hardtypes.NewQueryClient(grpcConn),
		Pricefeed: pricefeedtypes.NewQueryClient(grpcConn),
	}
}
