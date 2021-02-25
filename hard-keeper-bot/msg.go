package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
)

func CreateLiquidationMsgs(keeper sdk.AccAddress, borrowers Borrowers) []hardtypes.MsgLiquidate {
	msgs := make([]hardtypes.MsgLiquidate, len(borrowers))

	for index, borrower := range borrowers {
		msgs[index] = hardtypes.MsgLiquidate{
			Keeper:   keeper,
			Borrower: borrower,
		}
	}

	return msgs
}
