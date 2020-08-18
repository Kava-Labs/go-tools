package app

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/kava-labs/kava/app"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
)

type Client struct {
	restURL *url.URL
	codec   *codec.Codec
}

func NewClient(restURL string) Client {
	parsedURL, err := url.Parse(restURL)
	if err != nil {
		panic(err)
	}
	return Client{
		restURL: parsedURL,
		codec:   app.MakeCodec(),
	}
}

func (c Client) getAccount(address sdk.AccAddress) (authexported.Account, error) {

	queryPath, err := url.Parse(fmt.Sprintf("auth/accounts/%s", address))
	if err != nil {
		return nil, err
	}
	fullURL := c.restURL.ResolveReference(queryPath)

	resp, err := http.Get(fullURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed") // TODO better msg, rest.ErrorResponse
	}

	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res rest.ResponseWithHeight
	c.codec.MustUnmarshalJSON(bz, &res)
	var account authexported.Account
	c.codec.MustUnmarshalJSON(res.Result, &account)
	return account, nil
}

// func (c Client) broadcastTx(tx tmtypes.Tx) error {
// 	res, err := kc.kavaSDKClient.BroadcastTxSync(tx)
// 	if err != nil {
// 		return err
// 	}
// 	if res.Code != 0 { // tx failed to be submitted to the mempool
// 		return fmt.Errorf("transaction failed to get into mempool: %s", res.Log)
// 	}
// 	return nil
// }

// func (c Client) getChainID() (string, error) {
// 	infoResp, err := http.Get(kc.restURL + "/node_info")
// 	if err != nil {
// 		return "", err
// 	}
// 	defer infoResp.Body.Close()
// 	infoBz, err := ioutil.ReadAll(infoResp.Body)
// 	if err != nil {
// 		return "", err
// 	}
// 	var nodeInfo rpc.NodeInfoResponse
// 	kc.codec.MustUnmarshalJSON(infoBz, &nodeInfo)
// 	return nodeInfo.Network, nil
// }

func (c Client) getCDP(owner sdk.AccAddress, denom string) (cdptypes.CDP, error) {
	queryPath, err := url.Parse(fmt.Sprintf("cdp/cdps/cdp/%s/%s", owner, denom))
	if err != nil {
		return cdptypes.CDP{}, err
	}
	fullURL := c.restURL.ResolveReference(queryPath)

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
