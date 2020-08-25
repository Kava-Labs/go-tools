package app

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/rest"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/kava/app"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// Client handles querying data from and posting txs to a kava node.
type Client struct {
	restURL *url.URL
	codec   *codec.Codec
}

func NewClient(restURL string) (Client, error) {
	parsedURL, err := url.Parse(restURL)
	if err != nil {
		return Client{}, err
	}
	return Client{
		restURL: parsedURL,
		codec:   app.MakeCodec(),
	}, nil
}

func (c Client) getAccount(address sdk.AccAddress) (authexported.Account, int64, error) {
	var account authexported.Account
	var sdkResp rest.ResponseWithHeight
	err := c.fetchEndpoint(fmt.Sprintf("auth/accounts/%s", address), &sdkResp)
	if err != nil {
		return account, 0, err
	}
	err = c.codec.UnmarshalJSON(sdkResp.Result, &account)
	if err != nil {
		return account, 0, err
	}
	return account, sdkResp.Height, nil
}

func (c Client) getAugmentedCDP(owner sdk.AccAddress, denom string) (cdptypes.AugmentedCDP, int64, error) {
	var augmentedCDP cdptypes.AugmentedCDP
	var sdkResp rest.ResponseWithHeight
	err := c.fetchEndpoint(fmt.Sprintf("cdp/cdps/cdp/%s/%s", owner, denom), &sdkResp)
	if err != nil {
		return augmentedCDP, 0, err
	}
	err = c.codec.UnmarshalJSON(sdkResp.Result, &augmentedCDP)
	if err != nil {
		return augmentedCDP, 0, err
	}
	return augmentedCDP, sdkResp.Height, nil
}

func (c Client) getCDPParams() (cdptypes.Params, int64, error) {
	var cdpParams cdptypes.Params
	var sdkResp rest.ResponseWithHeight
	err := c.fetchEndpoint("cdp/parameters", &sdkResp)
	if err != nil {
		return cdpParams, 0, err
	}
	err = c.codec.UnmarshalJSON(sdkResp.Result, &cdpParams)
	if err != nil {
		return cdpParams, 0, err
	}
	return cdpParams, sdkResp.Height, nil
}

func (c Client) getTx(txHash []byte) (sdk.TxResponse, error) {
	var txResponse sdk.TxResponse
	err := c.fetchEndpoint(fmt.Sprintf("txs/%x", txHash), &txResponse)
	if err != nil {
		return txResponse, err
	}
	return txResponse, nil
}

func (c Client) getChainID() (string, error) {
	var nodeInfo rpc.NodeInfoResponse
	err := c.fetchEndpoint("node_info", &nodeInfo)
	if err != nil {
		return "", err
	}
	return nodeInfo.Network, nil
}

func (c Client) broadcastTx(stdTx authtypes.StdTx) error {
	// create post struct
	req := authrest.BroadcastReq{
		Tx:   stdTx,
		Mode: flags.BroadcastSync,
	}
	bz, err := c.codec.MarshalJSON(req)
	if err != nil {
		return err
	}
	// post
	postPath, err := url.Parse("txs")
	if err != nil {
		return err
	}
	fullURL := c.restURL.ResolveReference(postPath)

	res, err := http.Post(fullURL.String(), "application/json", bytes.NewBuffer(bz))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	bz, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", string(bz))
	}

	var txResponse sdk.TxResponse
	err = c.codec.UnmarshalJSON(bz, &txResponse)
	if err != nil {
		return err
	}
	if txResponse.Code != sdkerrors.SuccessABCICode {
		return NewMempoolRejectionError(txResponse.RawLog)
	}
	return nil
}

func (c Client) fetchEndpoint(path string, fetchTypePtr interface{}) error {
	queryPath, err := url.Parse(path)
	if err != nil {
		return err
	}
	fullURL := c.restURL.ResolveReference(queryPath)

	resp, err := http.Get(fullURL.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bz, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", string(bz))
	}

	err = c.codec.UnmarshalJSON(bz, fetchTypePtr)
	if err != nil {
		return err
	}
	return nil
}

func txHash(codec *codec.Codec, stdTx authtypes.StdTx) ([]byte, error) {
	bz, err := authtypes.DefaultTxEncoder(codec)(stdTx)
	if err != nil {
		return nil, err
	}
	return tmtypes.Tx(bz).Hash(), nil // TODO would this be neater without the tendermint dependency
}

type MempoolRejectionError struct {
	txLog string
}

func NewMempoolRejectionError(txLog string) *MempoolRejectionError {
	return &MempoolRejectionError{txLog: txLog}
}

func (mre *MempoolRejectionError) Error() string {
	return fmt.Sprintf("tx rejected from mempool: %s", mre.txLog)
}
