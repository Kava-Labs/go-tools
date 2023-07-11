package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BalanceType enum
type BalanceType string

const (
	BalanceTypeCDP  BalanceType = "cdp"
	BalanceTypeHard BalanceType = "hard"
	BalanceTypeEarn BalanceType = "earn"
)

// Balance defines the balance of a specific type
type Balance struct {
	Type   BalanceType
	Amount sdk.Coins
}

// AccountPositions defines the positions of an account
type AccountPositions struct {
	Address  sdk.AccAddress
	Balances []Balance
}
