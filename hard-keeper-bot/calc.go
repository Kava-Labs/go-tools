package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Borrowers []sdk.AccAddress

func GetBorrowersToLiquidate(data *LiquidationData) Borrowers {
	assets := data.Assets
	borrowersToLiquidate := Borrowers{}

	for _, pos := range data.Positions {
		totalBorrowableUSDAmount := sdk.ZeroDec()
		totalBorrowedUSDAmount := sdk.ZeroDec()

		for _, deposit := range pos.DepositedAmount {
			asset, ok := assets[deposit.Denom]
			if !ok {
				// if no asset, no market -- don't count towards borrowable amount
				continue
			}

			// usd value of deposit
			USDAmount := sdk.NewDecFromInt(deposit.Amount).Quo(sdk.NewDecFromInt(asset.ConversionFactor)).Mul(asset.Price)
			// borrowable usd value from deposit
			borrowableUSDAmount := USDAmount.Mul(asset.LoanToValueRatio)
			// add to total borrowable amount
			totalBorrowableUSDAmount = totalBorrowableUSDAmount.Add(borrowableUSDAmount)
		}

		for _, borrow := range pos.BorrowedAmount {
			asset, ok := assets[borrow.Denom]
			if !ok {
				// if no asset, no market -- don't count towards borrowed amount
				continue
			}

			// usd value of borrow
			USDAmount := sdk.NewDecFromInt(borrow.Amount).Quo(sdk.NewDecFromInt(asset.ConversionFactor)).Mul(asset.Price)
			// add to total borrowed amount
			totalBorrowedUSDAmount = totalBorrowedUSDAmount.Add(USDAmount)
		}

		// liquidate borrower if the have borrowed are over limit
		if totalBorrowedUSDAmount.GT(totalBorrowableUSDAmount) {
			borrowersToLiquidate = append(borrowersToLiquidate, pos.Address)
		}
	}

	return borrowersToLiquidate
}
