package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type GrpcClient struct {
	GrpcClientConn *grpc.ClientConn
	Tm             tmservice.ServiceClient
	Hard           hardtypes.QueryClient
	Pricefeed      pricefeedtypes.QueryClient
}

var _ LiquidationClient = (*GrpcClient)(nil)

func ctxAtHeight(height int64) context.Context {
	heightStr := strconv.FormatInt(height, 10)
	return metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, heightStr)
}

func NewGrpcClient(target string) GrpcClient {
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
		GrpcClientConn: grpcConn,
		Tm:             tmservice.NewServiceClient(grpcConn),
		Hard:           hardtypes.NewQueryClient(grpcConn),
		Pricefeed:      pricefeedtypes.NewQueryClient(grpcConn),
	}
}

func (c GrpcClient) GetInfo() (*InfoResponse, error) {
	latestBlock, err := c.Tm.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest block: %w", err)
	}

	return &InfoResponse{
		ChainId:      latestBlock.Block.Header.ChainID,
		LatestHeight: latestBlock.Block.Header.Height,
	}, nil
}

func (c GrpcClient) GetPrices(height int64) (pricefeedtypes.CurrentPrices, error) {
	pricesRes, err := c.Pricefeed.Prices(ctxAtHeight(height), &pricefeedtypes.QueryPricesRequest{})
	if err != nil {
		return nil, err
	}

	var prices []pricefeedtypes.CurrentPrice
	for _, response := range pricesRes.Prices {
		price := pricefeedtypes.CurrentPrice{
			MarketID: response.MarketID,
			Price:    response.Price,
		}
		prices = append(prices, price)
	}

	return prices, nil
}

func (c GrpcClient) GetMarkets(height int64) (hardtypes.MoneyMarkets, error) {
	paramsRes, err := c.Hard.Params(ctxAtHeight(height), &hardtypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return paramsRes.Params.MoneyMarkets, nil
}

func (c GrpcClient) GetBorrows(height int64) (hardtypes.Borrows, error) {
	borrowRes, err := c.Hard.Borrows(ctxAtHeight(height), &hardtypes.QueryBorrowsRequest{})
	if err != nil {
		return nil, err
	}

	var borrows []hardtypes.Borrow
	for _, response := range borrowRes.Borrows {
		borrow := hardtypes.Borrow{
			Borrower: sdk.AccAddress(response.Borrower),
			Amount:   response.Amount,
		}
		borrows = append(borrows, borrow)
	}

	return borrows, nil
}

func (c GrpcClient) GetDeposits(height int64) (hardtypes.Deposits, error) {
	depositRes, err := c.Hard.Deposits(ctxAtHeight(height), &hardtypes.QueryDepositsRequest{})
	if err != nil {
		return nil, err
	}

	var deposits []hardtypes.Deposit
	for _, response := range depositRes.Deposits {
		deposit := hardtypes.Deposit{
			Depositor: sdk.AccAddress(response.Depositor),
			Amount:    response.Amount,
		}
		deposits = append(deposits, deposit)
	}

	return deposits, nil
}
