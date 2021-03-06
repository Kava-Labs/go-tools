package claimer

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/kava/x/bep3"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type KavaClient struct {
	http *rpcclient.HTTP
	cdc  *codec.Codec
}

func NewKavaClient(cdc *codec.Codec, rpcAddr string, logger log.Logger) (*KavaClient, error) {
	http, err := rpcclient.New(rpcAddr, "/websocket")
	if err != nil {
		return nil, err
	}
	http.Logger = logger

	return &KavaClient{
		cdc:  cdc,
		http: http,
	}, nil
}

func (c *KavaClient) GetChainID() (string, error) {
	result, err := c.http.Status()
	if err != nil {
		return "", err
	}
	return result.NodeInfo.Network, nil
}

func (c *KavaClient) GetAccount(address sdk.AccAddress) (acc authexported.Account, err error) {
	params := authtypes.NewQueryAccountParams(address)
	bz, err := c.cdc.MarshalJSON(params)

	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("custom/acc/account/%s", address.String())

	result, err := c.ABCIQuery(path, bz)
	if err != nil {
		return nil, err
	}

	err = c.cdc.UnmarshalJSON(result, &acc)
	if err != nil {
		return nil, err
	}

	return acc, err
}

func (c *KavaClient) GetAtomicSwap(swapID []byte) (bep3.AtomicSwap, error) {
	params := bep3.NewQueryAtomicSwapByID(swapID)
	bz, err := c.cdc.MarshalJSON(params)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}

	result, err := c.ABCIQuery("custom/bep3/swap", bz)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}

	var swap bep3.AtomicSwap
	err = c.cdc.UnmarshalJSON(result, &swap)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}
	return swap, nil
}

func (c *KavaClient) ABCIQuery(path string, data tmbytes.HexBytes) ([]byte, error) {
	result, err := c.http.ABCIQuery(path, data)
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
	return c.http.BroadcastTxSync(tx)
}

func (c *KavaClient) BroadcastTxCommit(tx tmtypes.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	return c.http.BroadcastTxCommit(tx)
}

func (c *KavaClient) GetTxConfirmation(txHash []byte) (*ctypes.ResultTx, error) {
	return c.http.Tx(txHash, false)
}
