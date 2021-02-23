package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	kava "github.com/kava-labs/kava/app"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/p2p"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func init() {
	kavaConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(kavaConfig)
	kavaConfig.Seal()
}

// ABCIQueryDataFunc is used to mock value in query response
type ABCIQueryDataFunc func(data bytes.HexBytes) []byte

// ABCICheckFunc is used to verify path, params, and opts in tests
type ABCICheckFunc func(
	path string,
	params bytes.HexBytes,
	opts rpcclient.ABCIQueryOptions,
)

type MockRpcClient struct {
	StatusResponse        ctypes.ResultStatus
	StatusResponseErr     error
	ABCICheckFunc         ABCICheckFunc
	ABCIQueryDataFunc     ABCIQueryDataFunc
	ABCIResponseQueryCode uint32
	ABCIResponseQueryLog  string
	ABCIResponseErr       error
}

func (m *MockRpcClient) Status() (*ctypes.ResultStatus, error) {
	return &m.StatusResponse, m.StatusResponseErr
}

func (m *MockRpcClient) ABCIQueryWithOptions(
	path string,
	data bytes.HexBytes,
	opts rpcclient.ABCIQueryOptions,
) (*ctypes.ResultABCIQuery, error) {
	// allow test code to check path, data, opts globally
	m.ABCICheckFunc(path, data, opts)

	// build response and allow injection of return value
	return &ctypes.ResultABCIQuery{
		Response: abci.ResponseQuery{
			Code:  m.ABCIResponseQueryCode,
			Log:   m.ABCIResponseQueryLog,
			Value: m.ABCIQueryDataFunc(data),
		},
	}, m.ABCIResponseErr
}

func TestGetInfo(t *testing.T) {
	rpc := &MockRpcClient{}
	cdc := kava.MakeCodec()
	client := NewRpcAuctionClient(rpc, cdc)

	rpc.StatusResponse = ctypes.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			Network: "kava-5",
		},
		SyncInfo: ctypes.SyncInfo{
			LatestBlockHeight: 100,
		},
	}

	info, err := client.GetInfo()
	if err != nil {
		t.Fatalf("GetInfo failed with err %s", err)
	}

	assert.Equal(t, "kava-5", info.ChainId)
	assert.Equal(t, int64(100), info.LatestHeight)

	responseErr := errors.New("error getting status")
	rpc.StatusResponseErr = responseErr

	info, err = client.GetInfo()
	assert.Nil(t, info)
	assert.Equal(t, responseErr, err)
}

func TestGetPrices(t *testing.T) {
	rpc := &MockRpcClient{}
	cdc := kava.MakeCodec()
	client := NewRpcAuctionClient(rpc, cdc)

	height := int64(1001)

	prices := pricefeedtypes.CurrentPrices{
		pricefeedtypes.CurrentPrice{
			MarketID: "busd:usd",
			Price:    sdk.MustNewDecFromStr("1.004"),
		},
		pricefeedtypes.CurrentPrice{
			MarketID: "busd:usd:30",
			Price:    sdk.MustNewDecFromStr("1.003"),
		},
	}

	pricesData, err := cdc.MarshalJSON(&prices)
	assert.Nil(t, err)

	// check correct parameters are called
	rpc.ABCICheckFunc = func(
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) {
		// prices query
		assert.Equal(t, fmt.Sprintf("custom/%s/%s", pricefeedtypes.QuerierRoute, pricefeedtypes.QueryPrices), path)
		// no parameters
		assert.Equal(t, bytes.HexBytes{}, data)
		// queries at provided height
		assert.Equal(t, rpcclient.ABCIQueryOptions{Height: height, Prove: false}, opts)
	}

	// return prices data from abci query
	rpc.ABCIQueryDataFunc = func(data bytes.HexBytes) []byte {
		return pricesData
	}

	type getPricesTest struct {
		code uint32
		log  string
		err  error
		data pricefeedtypes.CurrentPrices
	}

	tests := []getPricesTest{
		{code: 0, log: "", err: nil, data: prices},
		{code: 0, log: "", err: errors.New("abci error"), data: nil},
		{code: 1, log: "argument error", err: nil, data: nil},
	}

	for _, tc := range tests {
		rpc.ABCIResponseQueryCode = tc.code
		rpc.ABCIResponseQueryLog = tc.log
		rpc.ABCIResponseErr = tc.err

		resp, err := client.GetPrices(height)

		if tc.log != "" {
			tc.err = errors.New(tc.log)
		}

		assert.Equal(t, tc.data, resp)
		assert.Equal(t, tc.err, err)
	}
}

func TestGetAuctions(t *testing.T) {
	rpc := &MockRpcClient{}
	cdc := kava.MakeCodec()
	client := NewRpcAuctionClient(rpc, cdc)

	height := int64(1001)

	lotReturns, _ := auctiontypes.NewWeightedAddresses([]sdk.AccAddress{sdk.AccAddress(crypto.AddressHash([]byte("testborrower")))}, []sdk.Int{sdk.NewInt(100)})

	auctions := auctiontypes.Auctions{
		auctiontypes.CollateralAuction{
			BaseAuction: auctiontypes.BaseAuction{
				ID:              1,
				Initiator:       "hard",
				Lot:             sdk.NewInt64Coin("ukava", 100000000),
				Bidder:          sdk.AccAddress{},
				Bid:             sdk.NewInt64Coin("usdx", 100000000),
				HasReceivedBids: false,
				EndTime:         time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
				MaxEndTime:      time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
			},
			CorrespondingDebt: sdk.NewInt64Coin("debt", 125000000),
			MaxBid:            sdk.NewInt64Coin("usdx", 120000000),
			LotReturns:        lotReturns,
		},
		auctiontypes.CollateralAuction{
			BaseAuction: auctiontypes.BaseAuction{
				ID:              2,
				Initiator:       "hard",
				Lot:             sdk.NewInt64Coin("ukava", 100000000),
				Bidder:          sdk.AccAddress{},
				Bid:             sdk.NewInt64Coin("usdx", 100000000),
				HasReceivedBids: false,
				EndTime:         time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
				MaxEndTime:      time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
			},
			CorrespondingDebt: sdk.NewInt64Coin("debt", 125000000),
			MaxBid:            sdk.NewInt64Coin("usdx", 120000000),
			LotReturns:        lotReturns,
		},
		auctiontypes.CollateralAuction{
			BaseAuction: auctiontypes.BaseAuction{
				ID:              3,
				Initiator:       "hard",
				Lot:             sdk.NewInt64Coin("ukava", 100000000),
				Bidder:          sdk.AccAddress{},
				Bid:             sdk.NewInt64Coin("usdx", 100000000),
				HasReceivedBids: false,
				EndTime:         time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
				MaxEndTime:      time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
			},
			CorrespondingDebt: sdk.NewInt64Coin("debt", 125000000),
			MaxBid:            sdk.NewInt64Coin("usdx", 120000000),
			LotReturns:        lotReturns,
		},
		auctiontypes.CollateralAuction{
			BaseAuction: auctiontypes.BaseAuction{
				ID:              4,
				Initiator:       "hard",
				Lot:             sdk.NewInt64Coin("ukava", 100000000),
				Bidder:          sdk.AccAddress{},
				Bid:             sdk.NewInt64Coin("usdx", 100000000),
				HasReceivedBids: false,
				EndTime:         time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
				MaxEndTime:      time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
			},
			CorrespondingDebt: sdk.NewInt64Coin("debt", 125000000),
			MaxBid:            sdk.NewInt64Coin("usdx", 120000000),
			LotReturns:        lotReturns,
		},
		auctiontypes.CollateralAuction{
			BaseAuction: auctiontypes.BaseAuction{
				ID:              5,
				Initiator:       "hard",
				Lot:             sdk.NewInt64Coin("ukava", 100000000),
				Bidder:          sdk.AccAddress{},
				Bid:             sdk.NewInt64Coin("usdx", 100000000),
				HasReceivedBids: false,
				EndTime:         time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
				MaxEndTime:      time.Date(2021, 02, 04, 14, 0, 0, 0, time.UTC),
			},
			CorrespondingDebt: sdk.NewInt64Coin("debt", 125000000),
			MaxBid:            sdk.NewInt64Coin("usdx", 120000000),
			LotReturns:        lotReturns,
		},
	}

	rpc.ABCICheckFunc = func(
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) {
		// auctions query
		assert.Equal(t, fmt.Sprintf("custom/%s/%s", auctiontypes.QuerierRoute, auctiontypes.QueryGetAuctions), path)
		// queries at provided height
		assert.Equal(t, rpcclient.ABCIQueryOptions{Height: height, Prove: false}, opts)

		// decode parameters
		var params auctiontypes.QueryAllAuctionParams
		err := cdc.UnmarshalJSON(data, &params)
		assert.Nil(t, err)

		// owner is not set -- we fetch all auctions
		assert.Equal(t, sdk.AccAddress{}, params.Owner)
		// type, denom, phase are not set -- we fetch all auctions
		assert.Equal(t, "", params.Type)
		assert.Equal(t, "", params.Denom)
		assert.Equal(t, "", params.Phase)
	}

	rpc.ABCIQueryDataFunc = func(data bytes.HexBytes) []byte {
		var params auctiontypes.QueryAllAuctionParams
		err := cdc.UnmarshalJSON(data, &params)
		assert.Nil(t, err)

		start := (params.Page - 1) * params.Limit
		end := start + params.Limit
		if end > len(auctions) {
			end = len(auctions)
		}

		auctionsPage := auctions[start:end]
		auctionsData, err := cdc.MarshalJSON(&auctionsPage)
		assert.Nil(t, err)

		return auctionsData
	}

	type getAuctionsTest struct {
		pageLimit int
		code      uint32
		log       string
		err       error
		data      auctiontypes.Auctions
	}

	tests := []getAuctionsTest{
		{pageLimit: 10, code: 0, log: "", err: nil, data: auctions},
		// {pageLimit: 1, code: 0, log: "", err: nil, data: auctions},
		// {pageLimit: 2, code: 0, log: "", err: nil, data: auctions},
		// {pageLimit: 5, code: 0, log: "", err: nil, data: auctions},
		// {code: 0, log: "", err: errors.New("abci error"), data: auctions},
		// {code: 1, log: "argument error", err: nil, data: nil},
	}

	for _, tc := range tests {
		rpc.ABCIResponseQueryCode = tc.code
		rpc.ABCIResponseQueryLog = tc.log
		rpc.ABCIResponseErr = tc.err

		if tc.pageLimit > 0 {
			client.PageLimit = tc.pageLimit
		}

		resp, err := client.GetAuctions(height)

		if tc.log != "" {
			tc.err = errors.New(tc.log)
		}

		assert.Equal(t, tc.data, resp)
		assert.Equal(t, tc.err, err)
	}
}
