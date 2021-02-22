package client

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmlog "github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type KavaClient struct {
	Http *rpcclient.HTTP
	Cdc  *codec.Codec
}

func NewKavaClient(cdc *codec.Codec, rpcAddr string, logger tmlog.Logger) (*KavaClient, error) {
	http, err := rpcclient.New(rpcAddr, "/websocket")
	if err != nil {
		return nil, err
	}
	http.Logger = logger

	return &KavaClient{
		Cdc:  cdc,
		Http: http,
	}, nil
}

func (c *KavaClient) GetChainID() (string, error) {
	result, err := c.Http.Status()
	if err != nil {
		return "", err
	}
	return result.NodeInfo.Network, nil
}

func (c *KavaClient) GetAccount(address sdk.AccAddress) (acc authexported.Account, err error) {
	params := authtypes.NewQueryAccountParams(address)
	bz, err := c.Cdc.MarshalJSON(params)

	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("custom/acc/account/%s", address.String())

	result, err := c.ABCIQuery(path, bz)
	if err != nil {
		return nil, err
	}

	err = c.Cdc.UnmarshalJSON(result, &acc)
	if err != nil {
		return nil, err
	}

	return acc, err
}

func (c *KavaClient) ABCIQuery(path string, data tmbytes.HexBytes) ([]byte, error) {
	result, err := c.Http.ABCIQuery(path, data)
	if err != nil {
		return []byte{}, err
	}

	resp := result.Response
	if !resp.IsOK() {
		return []byte{}, errors.New(resp.Log)
	}

	value := result.Response.GetValue()
	if len(value) == 0 {
		return []byte{}, nil
	}

	return value, nil
}

func (c *KavaClient) BroadcastTxSync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	return c.Http.BroadcastTxSync(tx)
}

func (c *KavaClient) BroadcastTxCommit(tx tmtypes.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	return c.Http.BroadcastTxCommit(tx)
}

func (c *KavaClient) GetTxConfirmation(txHash []byte) (*ctypes.ResultTx, error) {
	return c.Http.Tx(txHash, false)
}
