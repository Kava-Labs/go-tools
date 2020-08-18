package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/kava-labs/kava/app"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
)

func main() {

}

type Client struct {
	restURL string
	codec   *codec.Codec
}

func NewClient(restURL string) Client {
	return Client{
		restURL: restURL,
		codec:   app.MakeCodec(),
	}
}

func (c Client) GetCDP(owner sdk.AccAddress, denom string) (cdptypes.CDP, error) {
	restURL, err := url.Parse(c.restURL)
	if err != nil {
		return cdptypes.CDP{}, err
	}
	queryPath, err := url.Parse(fmt.Sprintf("cdp/cdps/cdp/%s/%s", owner, denom))
	if err != nil {
		return cdptypes.CDP{}, err
	}
	fullURL := restURL.ResolveReference(queryPath)

	resp, err := http.Get(fullURL.String())
	if err != nil {
		return cdptypes.CDP{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cdptypes.CDP{}, fmt.Errorf("request failed") // TODO better msg, rest.ErrorResponse
	}
	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return cdptypes.CDP{}, err
	}

	var res rest.ResponseWithHeight
	c.codec.MustUnmarshalJSON(bz, &res)
	var augmentedCDP cdptypes.AugmentedCDP
	c.codec.MustUnmarshalJSON(res.Result, &augmentedCDP)

	return augmentedCDP.CDP, nil
}
