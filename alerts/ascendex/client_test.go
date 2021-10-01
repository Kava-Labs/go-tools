package ascendex

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAscendexClient(t *testing.T) {
	client := NewAscendexHttpClient()
	summary, err := client.Ticker("USDX/USDT")
	require.NoError(t, err)

	assert.Equal(t, "USDX/USDT", summary.Symbol)

	values := []struct {
		Amount sdk.Dec
		Name   string
	}{
		{Amount: summary.Open, Name: "Open"},
		{Amount: summary.Close, Name: "Close"},
		{Amount: summary.High, Name: "High"},
		{Amount: summary.Low, Name: "Low"},
	}

	for _, v := range values {
		assert.True(t, v.Amount.GT(sdk.MustNewDecFromStr("0.5")) &&
			v.Amount.LT(sdk.MustNewDecFromStr("1.5")), "USDX/USDT %v should be between 0.5 - 1.5", v.Name)
	}
}

func TestAscendexMockClient(t *testing.T) {
	client := NewAscendexMockClient()
	summary, err := client.Ticker("USDX/USDT")
	require.NoError(t, err)

	assert.Equal(t, SymbolSummary{
		Symbol: "USDX/USDT",
		Open:   sdk.MustNewDecFromStr("0.9721"),
		Close:  sdk.MustNewDecFromStr("0.9617"),
		High:   sdk.MustNewDecFromStr("0.999"),
		Low:    sdk.MustNewDecFromStr("0.9321"),
		Volume: sdk.MustNewDecFromStr("168914"),
		Ask: []sdk.Dec{
			sdk.MustNewDecFromStr("0.9638"),
			sdk.MustNewDecFromStr("2852.2"),
		},
		Bid: []sdk.Dec{
			sdk.MustNewDecFromStr("0.9569"),
			sdk.MustNewDecFromStr("8.5"),
		},
	}, summary)
}
