package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Position struct {
	Address         sdk.AccAddress
	BorrowedAmount  sdk.Coins
	DepositedAmount sdk.Coins
}

type AssetData struct {
	Price            sdk.Dec
	LoanToValueRatio sdk.Dec
	ConversionFactor sdk.Int
}

type LiquidationData struct {
	Assets    map[string]AssetData
	Positions []Position
}

func GetLiquidationData(client LiquidationClient) (*LiquidationData, error) {
	// fetch chain info to get height
	info, err := client.GetInfo()
	if err != nil {
		return nil, err
	}

	// use height to get consistent state from rpc client
	height := info.LatestHeight

	markets, err := client.GetMarkets(height)
	if err != nil {
		return nil, err
	}

	prices, err := client.GetPrices(height)
	if err != nil {
		return nil, err
	}

	borrows, err := client.GetBorrows(height)
	if err != nil {
		return nil, err
	}

	deposits, err := client.GetDeposits(height)
	if err != nil {
		return nil, err
	}

	// map price data
	priceData := make(map[string]sdk.Dec)
	for _, price := range prices {
		priceData[price.MarketID] = price.Price
	}

	// loop markets and create AssetData
	assetData := make(map[string]AssetData)
	for _, market := range markets {
		price, ok := priceData[market.SpotMarketID]
		if !ok {
			return nil, fmt.Errorf("no price for market id %s", market.SpotMarketID)
		}

		assetData[market.Denom] = AssetData{
			Price:            price,
			LoanToValueRatio: market.BorrowLimit.LoanToValue,
			ConversionFactor: market.ConversionFactor,
		}
	}

	// loop deposits and map into lookup table by address
	depositData := make(map[string]sdk.Coins)
	for _, deposit := range deposits {
		depositData[deposit.Depositor.String()] = deposit.Amount
	}

	// loop through borrows and build position data
	// number of posistions matches number of borrows
	positions := make([]Position, len(borrows))

	for index, borrow := range borrows {
		addr := borrow.Borrower

		depositAmount, ok := depositData[addr.String()]
		if !ok {
			depositAmount = sdk.Coins{}
		}

		positions[index] = Position{
			Address:         addr,
			BorrowedAmount:  borrow.Amount,
			DepositedAmount: depositAmount,
		}
	}

	return &LiquidationData{
		Assets:    assetData,
		Positions: positions,
	}, nil
}
