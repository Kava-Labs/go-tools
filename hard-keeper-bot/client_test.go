package main

import (
	"context"
	"errors"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/p2p"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

func init() {
	app.SetSDKConfig()
}

// ABCIQueryDataFunc is used to mock value in query response
type ABCIQueryDataFunc func(data bytes.HexBytes) []byte

// ABCICheckFunc is used to verify path, params, and opts in tests
type ABCICheckFunc func(
	ctx context.Context,
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

func (m *MockRpcClient) Status(ctx context.Context) (*ctypes.ResultStatus, error) {
	return &m.StatusResponse, m.StatusResponseErr
}

func (m *MockRpcClient) ABCIQuery(
	ctx context.Context,
	path string,
	data bytes.HexBytes,
) (*ctypes.ResultABCIQuery, error) {
	// allow test code to check path, data, opts globally
	m.ABCICheckFunc(context.Background(), path, data, rpcclient.DefaultABCIQueryOptions)

	// build response and allow injection of return value
	return &ctypes.ResultABCIQuery{
		Response: abci.ResponseQuery{
			Code:  m.ABCIResponseQueryCode,
			Log:   m.ABCIResponseQueryLog,
			Value: m.ABCIQueryDataFunc(data),
		},
	}, m.ABCIResponseErr
}

func (m *MockRpcClient) ABCIQueryWithOptions(
	ctx context.Context,
	path string,
	data bytes.HexBytes,
	opts rpcclient.ABCIQueryOptions,
) (*ctypes.ResultABCIQuery, error) {
	// allow test code to check path, data, opts globally
	m.ABCICheckFunc(context.Background(), path, data, opts)

	// build response and allow injection of return value
	return &ctypes.ResultABCIQuery{
		Response: abci.ResponseQuery{
			Code:  m.ABCIResponseQueryCode,
			Log:   m.ABCIResponseQueryLog,
			Value: m.ABCIQueryDataFunc(data),
		},
	}, m.ABCIResponseErr
}

func (m *MockRpcClient) BroadcastTxSync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	return nil, nil
}

func TestGetInfo(t *testing.T) {
	rpc := &MockRpcClient{}
	cdc := app.MakeEncodingConfig().Amino
	client := NewRpcLiquidationClient(rpc, cdc)

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
	cdc := app.MakeEncodingConfig().Amino
	client := NewRpcLiquidationClient(rpc, cdc)

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
		ctx context.Context,
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

func TestGetMarkets(t *testing.T) {
	rpc := &MockRpcClient{}
	cdc := app.MakeEncodingConfig().Amino
	client := NewRpcLiquidationClient(rpc, cdc)

	height := int64(1001)

	params := hardtypes.Params{
		MoneyMarkets: hardtypes.MoneyMarkets{
			{
				Denom:        "busd",
				SpotMarketID: "busd:usd",
				BorrowLimit: hardtypes.BorrowLimit{
					HasMaxLimit:  false,
					MaximumLimit: sdk.MustNewDecFromStr("0.0"),
					LoanToValue:  sdk.MustNewDecFromStr("0.5"),
				},
				ConversionFactor: sdk.NewInt(10000000),
				InterestRateModel: hardtypes.InterestRateModel{
					BaseRateAPY:    sdk.MustNewDecFromStr("0.0"),
					BaseMultiplier: sdk.MustNewDecFromStr("0.10"),
					Kink:           sdk.MustNewDecFromStr("0.8"),
					JumpMultiplier: sdk.MustNewDecFromStr("1.0"),
				},
				ReserveFactor:          sdk.MustNewDecFromStr("1.0"),
				KeeperRewardPercentage: sdk.MustNewDecFromStr("0.02"),
			},
			{
				Denom:        "btcb",
				SpotMarketID: "btcb:usd",
				BorrowLimit: hardtypes.BorrowLimit{
					HasMaxLimit:  false,
					MaximumLimit: sdk.MustNewDecFromStr("0.0"),
					LoanToValue:  sdk.MustNewDecFromStr("0.5"),
				},
				ConversionFactor: sdk.NewInt(10000000),
				InterestRateModel: hardtypes.InterestRateModel{
					BaseRateAPY:    sdk.MustNewDecFromStr("0.0"),
					BaseMultiplier: sdk.MustNewDecFromStr("0.10"),
					Kink:           sdk.MustNewDecFromStr("0.8"),
					JumpMultiplier: sdk.MustNewDecFromStr("1.0"),
				},
				ReserveFactor:          sdk.MustNewDecFromStr("1.0"),
				KeeperRewardPercentage: sdk.MustNewDecFromStr("0.02"),
			},
		},
		MinimumBorrowUSDValue: sdk.MustNewDecFromStr("1000000"),
	}

	paramsData, err := cdc.MarshalJSON(&params)
	assert.Nil(t, err)

	// check correct parameters are called
	rpc.ABCICheckFunc = func(
		ctx context.Context,
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) {
		// prices query
		assert.Equal(t, fmt.Sprintf("custom/%s/%s", hardtypes.QuerierRoute, hardtypes.QueryGetParams), path)
		// no parameters
		assert.Equal(t, bytes.HexBytes{}, data)
		// queries at provided height
		assert.Equal(t, rpcclient.ABCIQueryOptions{Height: height, Prove: false}, opts)
	}

	// return prices data from abci query
	rpc.ABCIQueryDataFunc = func(data bytes.HexBytes) []byte {
		return paramsData
	}

	type getMarketsTest struct {
		code uint32
		log  string
		err  error
		data hardtypes.MoneyMarkets
	}

	tests := []getMarketsTest{
		{code: 0, log: "", err: nil, data: params.MoneyMarkets},
		{code: 0, log: "", err: errors.New("abci error"), data: nil},
		{code: 1, log: "argument error", err: nil, data: nil},
	}

	for _, tc := range tests {
		rpc.ABCIResponseQueryCode = tc.code
		rpc.ABCIResponseQueryLog = tc.log
		rpc.ABCIResponseErr = tc.err

		resp, err := client.GetMarkets(height)

		if tc.log != "" {
			tc.err = errors.New(tc.log)
		}

		assert.Equal(t, tc.data, resp)
		assert.Equal(t, tc.err, err)
	}
}

func TestGetBorrows(t *testing.T) {
	rpc := &MockRpcClient{}
	cdc := app.MakeEncodingConfig().Amino
	client := NewRpcLiquidationClient(rpc, cdc)

	height := int64(1001)

	mockCoins := sdk.Coins{{Amount: sdk.NewInt(100000), Denom: "busd"}, {Amount: sdk.NewInt(100000), Denom: "btcb"}}
	mockIndex := hardtypes.BorrowInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}}

	borrows := hardtypes.Borrows{
		{
			Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
			Amount:   mockCoins,
			Index:    mockIndex,
		},
		{
			Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
			Amount:   mockCoins,
			Index:    mockIndex,
		},
		{
			Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
			Amount:   mockCoins,
			Index:    mockIndex,
		},
		{
			Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower4"))),
			Amount:   mockCoins,
			Index:    mockIndex,
		},
		{
			Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower5"))),
			Amount:   mockCoins,
			Index:    mockIndex,
		},
	}

	// check correct parameters are called
	rpc.ABCICheckFunc = func(
		ctx context.Context,
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) {
		// prices query
		assert.Equal(t, fmt.Sprintf("custom/%s/%s", hardtypes.QuerierRoute, hardtypes.QueryGetBorrows), path)
		// queries at provided height
		assert.Equal(t, rpcclient.ABCIQueryOptions{Height: height, Prove: false}, opts)

		// decode parameters
		var params hardtypes.QueryBorrowsParams
		err := cdc.UnmarshalJSON(data, &params)
		assert.Nil(t, err)

		// owner is not set -- we fetch all borrows
		assert.Equal(t, sdk.AccAddress{}, params.Owner)
		// denom is not set -- we fetch all borrow denoms
		assert.Equal(t, "", params.Denom)
	}

	// return prices data from abci query
	rpc.ABCIQueryDataFunc = func(data bytes.HexBytes) []byte {
		var params hardtypes.QueryBorrowsParams
		err := cdc.UnmarshalJSON(data, &params)
		assert.Nil(t, err)

		start := (params.Page - 1) * params.Limit
		end := start + params.Limit

		if end > len(borrows) {
			end = len(borrows)
		}

		borrowsPage := borrows[start:end]
		borrowsData, err := cdc.MarshalJSON(&borrowsPage)
		assert.Nil(t, err)

		return borrowsData
	}

	type getBorrowsTest struct {
		pageLimit int
		code      uint32
		log       string
		err       error
		data      hardtypes.Borrows
	}

	tests := []getBorrowsTest{
		{pageLimit: 10, code: 0, log: "", err: nil, data: borrows}, // larger limit than results
		{pageLimit: 1, code: 0, log: "", err: nil, data: borrows},  // divisible limit
		{pageLimit: 2, code: 0, log: "", err: nil, data: borrows},  // non-divisible limit
		{pageLimit: 5, code: 0, log: "", err: nil, data: borrows},  // exact page
		{code: 0, log: "", err: errors.New("abci error"), data: nil},
		{code: 1, log: "argument error", err: nil, data: nil},
	}

	for _, tc := range tests {
		rpc.ABCIResponseQueryCode = tc.code
		rpc.ABCIResponseQueryLog = tc.log
		rpc.ABCIResponseErr = tc.err

		if tc.pageLimit > 0 {
			client.PageLimit = tc.pageLimit
		}

		resp, err := client.GetBorrows(height)

		if tc.log != "" {
			tc.err = errors.New(tc.log)
		}

		assert.Equal(t, tc.data, resp)
		assert.Equal(t, tc.err, err)
	}
}

func TestGetDeposits(t *testing.T) {
	rpc := &MockRpcClient{}
	cdc := app.MakeEncodingConfig().Amino
	client := NewRpcLiquidationClient(rpc, cdc)

	height := int64(1001)

	mockCoins := sdk.Coins{{Amount: sdk.NewInt(100000), Denom: "busd"}, {Amount: sdk.NewInt(100000), Denom: "btcb"}}
	mockIndex := hardtypes.SupplyInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}}

	deposits := hardtypes.Deposits{
		{
			Depositor: sdk.AccAddress(crypto.AddressHash([]byte("depositor1"))),
			Amount:    mockCoins,
			Index:     mockIndex,
		},
		{
			Depositor: sdk.AccAddress(crypto.AddressHash([]byte("depositor2"))),
			Amount:    mockCoins,
			Index:     mockIndex,
		},
		{
			Depositor: sdk.AccAddress(crypto.AddressHash([]byte("depositor3"))),
			Amount:    mockCoins,
			Index:     mockIndex,
		},
		{
			Depositor: sdk.AccAddress(crypto.AddressHash([]byte("depositor4"))),
			Amount:    mockCoins,
			Index:     mockIndex,
		},
		{
			Depositor: sdk.AccAddress(crypto.AddressHash([]byte("depositor5"))),
			Amount:    mockCoins,
			Index:     mockIndex,
		},
	}

	// check correct parameters are called
	rpc.ABCICheckFunc = func(
		ctx context.Context,
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) {
		// prices query
		assert.Equal(t, fmt.Sprintf("custom/%s/%s", hardtypes.QuerierRoute, hardtypes.QueryGetDeposits), path)
		// queries at provided height
		assert.Equal(t, rpcclient.ABCIQueryOptions{Height: height, Prove: false}, opts)

		// decode parameters
		var params hardtypes.QueryDepositsParams
		err := cdc.UnmarshalJSON(data, &params)
		assert.Nil(t, err)

		// owner is not set -- we fetch all deposits
		assert.Equal(t, sdk.AccAddress{}, params.Owner)
		// denom is not set -- we fetch all deposit denoms
		assert.Equal(t, "", params.Denom)
	}

	// return prices data from abci query
	rpc.ABCIQueryDataFunc = func(data bytes.HexBytes) []byte {
		var params hardtypes.QueryDepositsParams
		err := cdc.UnmarshalJSON(data, &params)
		assert.Nil(t, err)

		start := (params.Page - 1) * params.Limit
		end := start + params.Limit

		if end > len(deposits) {
			end = len(deposits)
		}

		depositsPage := deposits[start:end]
		depositsData, err := cdc.MarshalJSON(&depositsPage)
		assert.Nil(t, err)

		return depositsData
	}

	type getDepositsTest struct {
		pageLimit int
		code      uint32
		log       string
		err       error
		data      hardtypes.Deposits
	}

	tests := []getDepositsTest{
		{pageLimit: 10, code: 0, log: "", err: nil, data: deposits}, // larger limit than results
		{pageLimit: 1, code: 0, log: "", err: nil, data: deposits},  // divisible limit
		{pageLimit: 2, code: 0, log: "", err: nil, data: deposits},  // non-divisible limit
		{pageLimit: 5, code: 0, log: "", err: nil, data: deposits},  // exact page
		{code: 0, log: "", err: errors.New("abci error"), data: nil},
		{code: 1, log: "argument error", err: nil, data: nil},
	}

	for _, tc := range tests {
		rpc.ABCIResponseQueryCode = tc.code
		rpc.ABCIResponseQueryLog = tc.log
		rpc.ABCIResponseErr = tc.err

		if tc.pageLimit > 0 {
			client.PageLimit = tc.pageLimit
		}

		resp, err := client.GetDeposits(height)

		if tc.log != "" {
			tc.err = errors.New(tc.log)
		}

		assert.Equal(t, tc.data, resp)
		assert.Equal(t, tc.err, err)
	}
}
