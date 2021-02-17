package main

import (
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/crypto"
)

type MockClient struct {
	t              *testing.T
	ExpectedHeight int64
	InfoErr        error
	Prices         pricefeedtypes.CurrentPrices
	PricesErr      error
	Markets        hardtypes.MoneyMarkets
	MarketsErr     error
	Borrows        hardtypes.Borrows
	BorrowsErr     error
	Deposits       hardtypes.Deposits
	DepositsErr    error
}

func (m MockClient) GetInfo() (*InfoResponse, error) {
	return &InfoResponse{
		ChainId:      "kava-testnet",
		LatestHeight: m.ExpectedHeight,
	}, m.InfoErr
}

func (m MockClient) GetPrices(height int64) (pricefeedtypes.CurrentPrices, error) {
	m.checkHeight(height)
	return m.Prices, m.PricesErr
}

func (m MockClient) GetMarkets(height int64) (hardtypes.MoneyMarkets, error) {
	m.checkHeight(height)
	return m.Markets, m.MarketsErr
}

func (m MockClient) GetBorrows(height int64) (hardtypes.Borrows, error) {
	m.checkHeight(height)
	return m.Borrows, m.BorrowsErr
}

func (m MockClient) GetDeposits(height int64) (hardtypes.Deposits, error) {
	m.checkHeight(height)
	return m.Deposits, m.DepositsErr
}

func (m MockClient) checkHeight(height int64) {
	if m.ExpectedHeight != height {
		m.t.Fatalf("unexpected height %d", height)
	}
}

func TestGetPositionDataClientErrors(t *testing.T) {
	type getDataTest struct {
		client       MockClient
		expectedData PositionData
	}

	tests := []getDataTest{
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1001),
				InfoErr:        errors.New("error in info"),
			},
			expectedData: PositionData{},
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1002),
				PricesErr:      errors.New("error in get prices"),
			},
			expectedData: PositionData{},
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1003),
				MarketsErr:     errors.New("error in get markets"),
			},
			expectedData: PositionData{},
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1004),
				BorrowsErr:     errors.New("error in get borrows"),
			},
			expectedData: PositionData{},
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1005),
				DepositsErr:    errors.New("error in get deposits"),
			},
			expectedData: PositionData{},
		},
	}

	for _, tc := range tests {
		_, err := GetPositionData(tc.client)
		assert.NotNil(t, err)

		c := tc.client

		if c.InfoErr != nil {
			assert.Equal(t, c.InfoErr, err)
		}
		if c.PricesErr != nil {
			assert.Equal(t, c.PricesErr, err)
		}
		if c.MarketsErr != nil {
			assert.Equal(t, c.MarketsErr, err)
		}
		if c.BorrowsErr != nil {
			assert.Equal(t, c.BorrowsErr, err)
		}
		if c.DepositsErr != nil {
			assert.Equal(t, c.DepositsErr, err)
		}
	}
}

func TestGetLiquidationMissingData(t *testing.T) {
	type getDataTest struct {
		client       MockClient
		expectedData PositionData
		expectedErr  error
	}

	tests := []getDataTest{
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1001),
			}, // no data case
			expectedData: PositionData{
				Assets:    make(map[string]AssetInfo),
				Positions: []Position{},
			},
			expectedErr: nil,
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1002),
				Prices: pricefeedtypes.CurrentPrices{
					{
						MarketID: "busd:usd",
						Price:    sdk.MustNewDecFromStr("1.005"),
					},
					{
						MarketID: "btcb:usd:30",
						Price:    sdk.MustNewDecFromStr("50000.456123"),
					},
				},
			}, // prices and nothing else
			expectedData: PositionData{
				Assets:    make(map[string]AssetInfo),
				Positions: []Position{}, // no markets mean no data
			},
			expectedErr: nil,
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1002),
				Markets: hardtypes.MoneyMarkets{
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
				},
			}, // market but no prices
			expectedErr: errors.New("no price for market id busd:usd"),
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1002),
				Borrows: hardtypes.Borrows{
					{
						Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						Amount:   sdk.Coins{{Amount: sdk.NewInt(100000), Denom: "busd"}, {Amount: sdk.NewInt(100000), Denom: "btcb"}},
						Index:    hardtypes.BorrowInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}},
					},
					{
						Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
						Amount:   sdk.Coins{{Amount: sdk.NewInt(200000), Denom: "busd"}, {Amount: sdk.NewInt(200000), Denom: "btcb"}},
						Index:    hardtypes.BorrowInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.003")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.006")}},
					},
				},
			}, // borrows but nothing else
			expectedData: PositionData{
				Assets: make(map[string]AssetInfo),
				// we have positions if we have borrowers
				Positions: []Position{
					{
						Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(100000), Denom: "busd"}, {Amount: sdk.NewInt(100000), Denom: "btcb"}},
						DepositedAmount: sdk.Coins{},
					},
					{
						Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
						BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(200000), Denom: "busd"}, {Amount: sdk.NewInt(200000), Denom: "btcb"}},
						DepositedAmount: sdk.Coins{},
					},
				},
			},
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1002),
				Deposits: hardtypes.Deposits{
					{
						Depositor: sdk.AccAddress(crypto.AddressHash([]byte("depositor1"))),
						Amount:    sdk.Coins{{Amount: sdk.NewInt(100000), Denom: "busd"}, {Amount: sdk.NewInt(100000), Denom: "btcb"}},
						Index:     hardtypes.SupplyInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}},
					},
					{
						Depositor: sdk.AccAddress(crypto.AddressHash([]byte("depositor2"))),
						Amount:    sdk.Coins{{Amount: sdk.NewInt(200000), Denom: "busd"}, {Amount: sdk.NewInt(200000), Denom: "btcb"}},
						Index:     hardtypes.SupplyInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.003")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.006")}},
					},
				},
			}, // borrows but nothing else
			expectedData: PositionData{
				Assets: make(map[string]AssetInfo),
				// only borrowers count towards positions, depositors only are ignored
				Positions: []Position{},
			},
		},
	}

	for _, tc := range tests {
		data, err := GetPositionData(tc.client)
		assert.Equal(t, tc.expectedErr, err)
		if err == nil {
			assert.Equal(t, &tc.expectedData, data)
		} else {
			// error means nil return, no data
			assert.Equal(t, &tc.expectedData, &PositionData{})
		}
	}
}

func TestGetLiquidationAllData(t *testing.T) {
	type getDataTest struct {
		client       MockClient
		expectedData PositionData
	}

	tests := []getDataTest{
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1001),
				Prices: pricefeedtypes.CurrentPrices{
					{
						MarketID: "busd:usd",
						Price:    sdk.MustNewDecFromStr("1.0"),
					},
					{
						MarketID: "busd:usd:30",
						Price:    sdk.MustNewDecFromStr("2.0"),
					},
				},
				Markets: hardtypes.MoneyMarkets{
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
				},
				Borrows: hardtypes.Borrows{
					{
						Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						Amount:   sdk.Coins{{Amount: sdk.NewInt(1000000000), Denom: "busd"}},
						Index:    hardtypes.BorrowInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}},
					},
				},
				Deposits: hardtypes.Deposits{
					{
						Depositor: sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						Amount:    sdk.Coins{{Amount: sdk.NewInt(2000000000), Denom: "busd"}},
						Index:     hardtypes.SupplyInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}},
					},
				},
			}, // single market and single borrower
			expectedData: PositionData{
				Assets: map[string]AssetInfo{
					"busd": {
						Price:            sdk.MustNewDecFromStr("1.0"),
						LoanToValueRatio: sdk.MustNewDecFromStr("0.5"),
						ConversionFactor: sdk.NewInt(10000000),
					},
				},
				Positions: []Position{
					{
						Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(1000000000), Denom: "busd"}},
						DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(2000000000), Denom: "busd"}},
					},
				},
			},
		},
		{
			client: MockClient{
				t:              t,
				ExpectedHeight: int64(1001),
				Prices: pricefeedtypes.CurrentPrices{
					{
						MarketID: "busd:usd",
						Price:    sdk.MustNewDecFromStr("1.0"),
					},
					{
						MarketID: "busd:usd:30",
						Price:    sdk.MustNewDecFromStr("2.0"),
					},
					{
						MarketID: "btcb:usd",
						Price:    sdk.MustNewDecFromStr("50000.623453"),
					},
					{
						MarketID: "btcb:usd:30",
						Price:    sdk.MustNewDecFromStr("49000.125634"),
					},
				},
				Markets: hardtypes.MoneyMarkets{
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
						SpotMarketID: "btcb:usd:30",
						BorrowLimit: hardtypes.BorrowLimit{
							HasMaxLimit:  false,
							MaximumLimit: sdk.MustNewDecFromStr("0.0"),
							LoanToValue:  sdk.MustNewDecFromStr("0.4"),
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
				Borrows: hardtypes.Borrows{
					{
						Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						Amount:   sdk.Coins{{Amount: sdk.NewInt(1000000000), Denom: "busd"}},
						Index:    hardtypes.BorrowInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}},
					},
					{
						Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
						Amount:   sdk.Coins{{Amount: sdk.NewInt(300000000), Denom: "btcb"}},
						Index:    hardtypes.BorrowInterestFactors{{Denom: "btcb", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.003")}},
					},
					{
						Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
						Amount:   sdk.Coins{{Amount: sdk.NewInt(1000000000), Denom: "busd"}, {Amount: sdk.NewInt(300000000), Denom: "btcb"}},
						Index:    hardtypes.BorrowInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.008")}},
					},
				},
				Deposits: hardtypes.Deposits{
					{
						Depositor: sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						Amount:    sdk.Coins{{Amount: sdk.NewInt(2000000000), Denom: "busd"}},
						Index:     hardtypes.SupplyInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.004")}},
					},
					{
						Depositor: sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
						Amount:    sdk.Coins{{Amount: sdk.NewInt(600000000), Denom: "btcb"}},
						Index:     hardtypes.SupplyInterestFactors{{Denom: "btcb", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.003")}},
					},
					{
						Depositor: sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
						Amount:    sdk.Coins{{Amount: sdk.NewInt(2000000000), Denom: "busd"}, {Amount: sdk.NewInt(600000000), Denom: "btcb"}},
						Index:     hardtypes.SupplyInterestFactors{{Denom: "busd", Value: sdk.MustNewDecFromStr("1.002")}, {Denom: "btcb", Value: sdk.MustNewDecFromStr("1.008")}},
					},
				},
			}, // multiple markets, multiple borrowers
			expectedData: PositionData{
				Assets: map[string]AssetInfo{
					"busd": {
						Price:            sdk.MustNewDecFromStr("1.0"),
						LoanToValueRatio: sdk.MustNewDecFromStr("0.5"),
						ConversionFactor: sdk.NewInt(10000000),
					},
					"btcb": {
						Price:            sdk.MustNewDecFromStr("49000.125634"),
						LoanToValueRatio: sdk.MustNewDecFromStr("0.4"),
						ConversionFactor: sdk.NewInt(10000000),
					},
				},
				Positions: []Position{
					{
						Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
						BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(1000000000), Denom: "busd"}},
						DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(2000000000), Denom: "busd"}},
					},
					{
						Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
						BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(300000000), Denom: "btcb"}},
						DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(600000000), Denom: "btcb"}},
					},
					{
						Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
						BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(1000000000), Denom: "busd"}, {Amount: sdk.NewInt(300000000), Denom: "btcb"}},
						DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(2000000000), Denom: "busd"}, {Amount: sdk.NewInt(600000000), Denom: "btcb"}},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		data, err := GetPositionData(tc.client)
		assert.Nil(t, err)
		assert.Equal(t, &tc.expectedData, data)
	}
}
