package ascendex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ascendexBaseUrl = "https://ascendex.com/api/pro/v1"

type AscendexClient interface {
	Ticker(symbol string) (SymbolSummary, error)
}

type SymbolSummaryResponse struct {
	Code int64         `json:"code"`
	Data SymbolSummary `json:"data"`
}

type SymbolSummary struct {
	Symbol string    `json:"symbol"`
	Open   sdk.Dec   `json:"open,string"`
	Close  sdk.Dec   `json:"close,string"`
	High   sdk.Dec   `json:"high,string"`
	Low    sdk.Dec   `json:"low,string"`
	Volume sdk.Dec   `json:"volume,string"`
	Ask    []sdk.Dec `json:"ask"`
	Bid    []sdk.Dec `json:"bid"`
}

type AscendexHttpClient struct {
	client *http.Client
}

var _ AscendexClient = (*AscendexHttpClient)(nil)

func NewAscendexHttpClient() AscendexHttpClient {
	return AscendexHttpClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *AscendexHttpClient) Ticker(symbol string) (SymbolSummary, error) {
	res, err := c.client.Get(fmt.Sprintf("%v/ticker?symbol=%s", _ascendexBaseUrl, symbol))
	if err != nil {
		return SymbolSummary{}, err
	}
	defer res.Body.Close()

	var summaryRes SymbolSummaryResponse
	if err := json.NewDecoder(res.Body).Decode(&summaryRes); err != nil {
		return SymbolSummary{}, err
	}

	return summaryRes.Data, nil
}
