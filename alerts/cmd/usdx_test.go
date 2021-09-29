package cmd

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestExceedsDeviation(t *testing.T) {
	usdxBase := sdk.MustNewDecFromStr("1.0")

	tests := []struct {
		giveValue     sdk.Dec
		giveBase      sdk.Dec
		giveDeviation sdk.Dec
		wantExceeded  bool
	}{
		{sdk.MustNewDecFromStr("1.0"), usdxBase, sdk.MustNewDecFromStr("0.25"), false},
		{sdk.MustNewDecFromStr("1.0"), usdxBase, sdk.MustNewDecFromStr("0.1"), false},
		{sdk.MustNewDecFromStr("1.0"), usdxBase, sdk.MustNewDecFromStr("0.01"), false},
		// GTE is true
		{sdk.MustNewDecFromStr("1.1"), usdxBase, sdk.MustNewDecFromStr("0.1"), true},
		{sdk.MustNewDecFromStr("0.9"), usdxBase, sdk.MustNewDecFromStr("0.1"), true},
		{sdk.MustNewDecFromStr("1.3"), usdxBase, sdk.MustNewDecFromStr("0.25"), true},
		{sdk.MustNewDecFromStr("0.7"), usdxBase, sdk.MustNewDecFromStr("0.25"), true},

		{sdk.MustNewDecFromStr("1.2"), usdxBase, sdk.MustNewDecFromStr("0.25"), false},
		{sdk.MustNewDecFromStr("0.8"), usdxBase, sdk.MustNewDecFromStr("0.25"), false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v-%v", tt.giveValue, tt.giveDeviation), func(t *testing.T) {
			exceeded := exceedsDeviation(tt.giveValue, usdxBase, tt.giveDeviation)

			assert.Equal(t, tt.wantExceeded, exceeded)
		})
	}
}
