package main

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/query"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	PageLimit = 1000
)

type GrpcClient struct {
	cdc            codec.Codec
	GrpcClientConn *grpc.ClientConn
	Auth           authtypes.QueryClient
	Tx             txtypes.ServiceClient
	Tm             tmservice.ServiceClient
	Auction        auctiontypes.QueryClient
	Cdp            cdptypes.QueryClient
	Hard           hardtypes.QueryClient
	Pricefeed      pricefeedtypes.QueryClient
}

func NewGrpcClient(target string, enableTls bool, cdc codec.Codec) GrpcClient {
	var secureOpt grpc.DialOption
	if enableTls {
		creds := credentials.NewTLS(&tls.Config{})
		secureOpt = grpc.WithTransportCredentials(creds)
	} else {
		secureOpt = grpc.WithInsecure()
	}

	grpcConn, err := grpc.Dial(target, secureOpt)
	if err != nil {
		panic(err)
	}

	return GrpcClient{
		cdc:            cdc,
		GrpcClientConn: grpcConn,
		Auth:           authtypes.NewQueryClient(grpcConn),
		Tm:             tmservice.NewServiceClient(grpcConn),
		Tx:             txtypes.NewServiceClient(grpcConn),
		Auction:        auctiontypes.NewQueryClient(grpcConn),
		Cdp:            cdptypes.NewQueryClient(grpcConn),
		Hard:           hardtypes.NewQueryClient(grpcConn),
		Pricefeed:      pricefeedtypes.NewQueryClient(grpcConn),
	}
}

func (c *GrpcClient) LatestHeight() (int64, error) {
	latestBlock, err := c.Tm.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		return 0, err
	}

	return latestBlock.Block.Header.Height, nil
}

func (c *GrpcClient) ChainID() (string, error) {
	latestBlock, err := c.Tm.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		return "", err
	}

	return latestBlock.Block.Header.ChainID, nil
}

func (c *GrpcClient) Account(addr string) (authtypes.AccountI, error) {
	res, err := c.Auth.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: addr,
	})
	if err != nil {
		return nil, err
	}

	var acc authtypes.AccountI
	err = c.cdc.UnpackAny(res.Account, &acc)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (c *GrpcClient) BaseAccount(addr string) (authtypes.BaseAccount, error) {
	acc, err := c.Account(addr)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	bAcc, ok := acc.(*authtypes.BaseAccount)
	if !ok {
		return authtypes.BaseAccount{}, errors.New("not a base account")
	}

	return *bAcc, nil
}

func (c *GrpcClient) AllAuctions(ctx context.Context) ([]auctiontypes.Auction, error) {
	var key []byte
	var auctions []auctiontypes.Auction

	for {
		auctionsRes, err := c.Auction.Auctions(ctx, &auctiontypes.QueryAuctionsRequest{
			Pagination: &query.PageRequest{
				// Default limit if Limit is not provided is 100
				Limit: PageLimit,
				Key:   key,
			},
		})
		if err != nil {
			return nil, err
		}

		for _, anyAuction := range auctionsRes.Auction {
			var auction auctiontypes.Auction
			if err = c.cdc.UnpackAny(anyAuction, &auction); err != nil {
				return nil, err
			}

			auctions = append(auctions, auction)
		}

		key = auctionsRes.Pagination.NextKey

		if len(auctionsRes.Auction) < PageLimit {
			return auctions, nil
		}
	}
}
