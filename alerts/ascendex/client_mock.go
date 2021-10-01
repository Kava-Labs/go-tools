package ascendex

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _mockTickers = map[string]SymbolSummary{
	"USDX/USDT": {
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
	},
}

type AscendexMockClient struct{}

var _ AscendexClient = (*AscendexMockClient)(nil)

func NewAscendexMockClient() AscendexMockClient {
	return AscendexMockClient{}
}

func (c *AscendexMockClient) Ticker(symbol string) (SymbolSummary, error) {
	ticker, ok := _mockTickers[strings.ToUpper(symbol)]
	if !ok {
		return SymbolSummary{}, fmt.Errorf("could not find symbol ticker %s", symbol)
	}

	return ticker, nil
}
