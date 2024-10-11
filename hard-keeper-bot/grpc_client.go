package main

import (
	"context"
	"crypto/tls"
	"fmt"
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
	Conn            *grpc.ClientConn
	TmClient        tmservice.ServiceClient
	HardClient      hardtypes.QueryClient
	PricefeedClient pricefeedtypes.QueryClient
}

var _ LiquidationClient = (*GrpcClient)(nil)

func ctxAtHeight(height int64) context.Context {
	heightStr := strconv.FormatInt(height, 10)
	return metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, heightStr)
}

func NewGrpcClient(target string) (*GrpcClient, error) {
	grpcURL, err := url.Parse(target)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	var dialOptions grpc.DialOption
	switch grpcURL.Scheme {
	case "http":
		dialOptions = grpc.WithInsecure()
	case "https":
		dialOptions = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", grpcURL.Scheme)
	}

	conn, err := grpc.Dial(grpcURL.Host, dialOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &GrpcClient{
		Conn:            conn,
		TmClient:        tmservice.NewServiceClient(conn),
		HardClient:      hardtypes.NewQueryClient(conn),
		PricefeedClient: pricefeedtypes.NewQueryClient(conn),
	}, nil
}

func (c *GrpcClient) GetInfo() (*InfoResponse, error) {
	latestBlock, err := c.TmClient.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest block: %w", err)
	}

	return &InfoResponse{
		ChainId:      latestBlock.Block.Header.ChainID,
		LatestHeight: latestBlock.Block.Header.Height,
	}, nil
}

func (c *GrpcClient) GetPrices(height int64) (pricefeedtypes.CurrentPrices, error) {
	pricesRes, err := c.PricefeedClient.Prices(ctxAtHeight(height), &pricefeedtypes.QueryPricesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch prices: %w", err)
	}

	prices := make([]pricefeedtypes.CurrentPrice, len(pricesRes.Prices))
	for i, response := range pricesRes.Prices {
		prices[i] = pricefeedtypes.CurrentPrice{
			MarketID: response.MarketID,
			Price:    response.Price,
		}
	}

	return prices, nil
}

func (c *GrpcClient) GetMarkets(height int64) (hardtypes.MoneyMarkets, error) {
	paramsRes, err := c.HardClient.Params(ctxAtHeight(height), &hardtypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch money markets: %w", err)
	}

	return paramsRes.Params.MoneyMarkets, nil
}

func (c *GrpcClient) GetBorrows(height int64) (hardtypes.Borrows, error) {
	borrowRes, err := c.HardClient.Borrows(ctxAtHeight(height), &hardtypes.QueryBorrowsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch borrows: %w", err)
	}

	borrows := make([]hardtypes.Borrow, len(borrowRes.Borrows))
	for i, response := range borrowRes.Borrows {
		borrows[i] = hardtypes.Borrow{
			Borrower: sdk.AccAddress(response.Borrower),
			Amount:   response.Amount,
		}
	}

	return borrows, nil
}

func (c *GrpcClient) GetDeposits(height int64) (hardtypes.Deposits, error) {
	depositRes, err := c.HardClient.Deposits(ctxAtHeight(height), &hardtypes.QueryDepositsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deposits: %w", err)
	}

	deposits := make([]hardtypes.Deposit, len(depositRes.Deposits))
	for i, response := range depositRes.Deposits {
		deposits[i] = hardtypes.Deposit{
			Depositor: sdk.AccAddress(response.Depositor),
			Amount:    response.Amount,
		}
	}

	return deposits, nil
}
