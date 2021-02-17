package main

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/crypto"
)

func TestCreateLiquidationMsgs(t *testing.T) {
	type testLiquidationMsgs struct {
		keeper       sdk.AccAddress
		borrowers    Borrowers
		expectedMsgs []hardtypes.MsgLiquidate
	}

	tests := []testLiquidationMsgs{
		{
			keeper:       sdk.AccAddress(crypto.AddressHash([]byte("keeper"))),
			borrowers:    Borrowers{},
			expectedMsgs: []hardtypes.MsgLiquidate{},
		},
		{
			keeper:    sdk.AccAddress(crypto.AddressHash([]byte("keeper"))),
			borrowers: Borrowers{sdk.AccAddress(crypto.AddressHash([]byte("borrower")))},
			expectedMsgs: []hardtypes.MsgLiquidate{
				{
					Keeper:   sdk.AccAddress(crypto.AddressHash([]byte("keeper"))),
					Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower"))),
				},
			},
		},
		{
			keeper: sdk.AccAddress(crypto.AddressHash([]byte("keeper"))),
			borrowers: Borrowers{
				sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
				sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
				sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
			},
			expectedMsgs: []hardtypes.MsgLiquidate{
				{
					Keeper:   sdk.AccAddress(crypto.AddressHash([]byte("keeper"))),
					Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower1"))),
				},
				{
					Keeper:   sdk.AccAddress(crypto.AddressHash([]byte("keeper"))),
					Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower2"))),
				},
				{
					Keeper:   sdk.AccAddress(crypto.AddressHash([]byte("keeper"))),
					Borrower: sdk.AccAddress(crypto.AddressHash([]byte("borrower3"))),
				},
			},
		},
	}

	for _, tc := range tests {
		msgs := CreateLiquidationMsgs(tc.keeper, tc.borrowers)
		assert.Equal(t, tc.expectedMsgs, msgs)
	}
}
