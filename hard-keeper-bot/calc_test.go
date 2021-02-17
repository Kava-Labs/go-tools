package main

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/crypto"
)

func TestGetBorrowersToLiquidateNoData(t *testing.T) {
	data := &PositionData{}
	borrowers := GetBorrowersToLiquidate(data)
	assert.Equal(t, borrowers, Borrowers{})
}

func TestGetBorrowersToLiquidateMissingMarket(t *testing.T) {
	data := &PositionData{
		// no markets
		Assets: map[string]AssetInfo{},
		Positions: []Position{
			{
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(200000), Denom: "busd"}, {Amount: sdk.NewInt(200000), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(200000), Denom: "busd"}, {Amount: sdk.NewInt(200000), Denom: "btcb"}},
			},
		},
	}

	borrowers := GetBorrowersToLiquidate(data)
	assert.Equal(t, borrowers, Borrowers{})
}

func TestGetBorrowersToLiquidateSingleAssets(t *testing.T) {
	data := &PositionData{
		Assets: map[string]AssetInfo{
			"busd": {
				Price:            sdk.MustNewDecFromStr("1.0"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.5"),
				ConversionFactor: sdk.NewInt(100000000),
			},
			"btcb": {
				Price:            sdk.MustNewDecFromStr("50000.000000"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.4"),
				ConversionFactor: sdk.NewInt(100000000),
			},
			"usdx": {
				Price:            sdk.MustNewDecFromStr("1.0"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.8"),
				ConversionFactor: sdk.NewInt(1000000),
			},
		},
		Positions: []Position{
			{
				// OK: don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(0), Denom: "busd"}, {Amount: sdk.NewInt(0), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "busd"}, {Amount: sdk.NewInt(100000000), Denom: "btcb"}},
			},
			{
				// OK: busd only - exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(5000000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "busd"}},
			},
			{
				// NOT OK: busd only - over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower4"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(5000000001), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "busd"}},
			},
			{
				// OK: btcb only - exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower5"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(4000000000), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "btcb"}},
			},
			{
				// NOT OK: btcb only - over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower6"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(4000000001), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "btcb"}},
			},
			{
				// OK: usdx only - exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower7"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(8000000000), Denom: "usdx"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "usdx"}},
			},
			{
				// NOT OK: usdx only - over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower8"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(8000000001), Denom: "usdx"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "usdx"}},
			},
		},
	}

	borrowers := GetBorrowersToLiquidate(data)
	assert.Equal(t, borrowers, Borrowers{
		sdk.AccAddress(crypto.AddressHash([]byte("borrower4"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower6"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower8"))),
	})
}

func TestGetBorrowersToLiquidateDifferentAsset(t *testing.T) {
	data := &PositionData{
		Assets: map[string]AssetInfo{
			"busd": {
				Price:            sdk.MustNewDecFromStr("1.0"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.5"),
				ConversionFactor: sdk.NewInt(100000000),
			},
			"btcb": {
				Price:            sdk.MustNewDecFromStr("50000.000000"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.4"),
				ConversionFactor: sdk.NewInt(100000000),
			},
			"usdx": {
				Price:            sdk.MustNewDecFromStr("1.0"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.8"),
				ConversionFactor: sdk.NewInt(1000000),
			},
		},
		Positions: []Position{
			{
				// below limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(10000), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "busd"}},
			},
			{
				// exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(100000), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "busd"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(100001), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "busd"}},
			},
			{
				// below limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower4"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(2000000000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "btcb"}},
			},
			{
				// exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower5"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(200000000000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "btcb"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower6"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(200000000000001), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "btcb"}},
			},
			{
				// below limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower7"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(800000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(100000000), Denom: "usdx"}},
			},
			{
				// exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower8"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(8000000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(100000000), Denom: "usdx"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower9"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(8000000001), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(100000000), Denom: "usdx"}},
			},
			{
				// below limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower10"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(1600000), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "usdx"}},
			},
			{
				// exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower11"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(16000000), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "usdx"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower12"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(16000001), Denom: "btcb"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(10000000000), Denom: "usdx"}},
			},
		},
	}

	borrowers := GetBorrowersToLiquidate(data)
	assert.Equal(t, borrowers, Borrowers{
		sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower6"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower9"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower12"))),
	})
}

func TestGetBorrowersToLiquidateMultiAsset(t *testing.T) {
	data := &PositionData{
		Assets: map[string]AssetInfo{
			"busd": {
				Price:            sdk.MustNewDecFromStr("1.0"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.5"),
				ConversionFactor: sdk.NewInt(100000000),
			},
			"btcb": {
				Price:            sdk.MustNewDecFromStr("50000.000000"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.4"),
				ConversionFactor: sdk.NewInt(100000000),
			},
			"usdx": {
				Price:            sdk.MustNewDecFromStr("1.0"),
				LoanToValueRatio: sdk.MustNewDecFromStr("0.8"),
				ConversionFactor: sdk.NewInt(1000000),
			},
		},
		Positions: []Position{
			{
				// below limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(80000000), Denom: "btcb"}, {Amount: sdk.NewInt(4000000000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(1000000000000), Denom: "usdx"}},
			},
			{
				// exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(800000000), Denom: "btcb"}, {Amount: sdk.NewInt(40000000000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(1000000000000), Denom: "usdx"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(800000001), Denom: "btcb"}, {Amount: sdk.NewInt(40000000000000), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(1000000000000), Denom: "usdx"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower4"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(800000000), Denom: "btcb"}, {Amount: sdk.NewInt(40000000000001), Denom: "busd"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(1000000000000), Denom: "usdx"}},
			},
			{
				// below limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower5"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(2000000), Denom: "btcb"}, {Amount: sdk.NewInt(1500000000), Denom: "usdx"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(100000000), Denom: "btcb"}, {Amount: sdk.NewInt(10000000000000), Denom: "busd"}},
			},
			{
				// exact limit, don't liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower6"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(20000000), Denom: "btcb"}, {Amount: sdk.NewInt(15000000000), Denom: "usdx"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(100000000), Denom: "btcb"}, {Amount: sdk.NewInt(1000000000000), Denom: "busd"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower7"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(20000001), Denom: "btcb"}, {Amount: sdk.NewInt(15000000000), Denom: "usdx"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(100000000), Denom: "btcb"}, {Amount: sdk.NewInt(1000000000000), Denom: "busd"}},
			},
			{
				// over limit, liquidate
				Address:         sdk.AccAddress(crypto.AddressHash([]byte("borrower8"))),
				BorrowedAmount:  sdk.Coins{{Amount: sdk.NewInt(20000000), Denom: "btcb"}, {Amount: sdk.NewInt(15000000001), Denom: "usdx"}},
				DepositedAmount: sdk.Coins{{Amount: sdk.NewInt(100000000), Denom: "btcb"}, {Amount: sdk.NewInt(1000000000000), Denom: "busd"}},
			},
		},
	}

	borrowers := GetBorrowersToLiquidate(data)
	assert.Equal(t, borrowers, Borrowers{
		sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower4"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower7"))),
		sdk.AccAddress(crypto.AddressHash([]byte("borrower8"))),
	})
}
