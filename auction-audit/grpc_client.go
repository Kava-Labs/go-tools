package main

import (
	"crypto/tls"
	"log"
	"net/url"

	sdkClient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GrpcClient struct {
	cdc            codec.Codec
	Decoder        sdkClient.TxConfig
	GrpcClientConn *grpc.ClientConn
	Tm             tmservice.ServiceClient
	Auction        auctiontypes.QueryClient
	Tx             txtypes.ServiceClient
	CDP            cdptypes.QueryClient
	Pricefeed      pricefeedtypes.QueryClient
}

func NewGrpcClient(target string, cdc codec.Codec, txConfig sdkClient.TxConfig) GrpcClient {
	grpcUrl, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
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
		panic(err)
	}

	return GrpcClient{
		cdc:            cdc,
		Decoder:        txConfig,
		GrpcClientConn: grpcConn,
		Tm:             tmservice.NewServiceClient(grpcConn),
		Auction:        auctiontypes.NewQueryClient(grpcConn),
		Tx:             txtypes.NewServiceClient(grpcConn),
		CDP:            cdptypes.NewQueryClient(grpcConn),
		Pricefeed:      pricefeedtypes.NewQueryClient(grpcConn),
	}
}
