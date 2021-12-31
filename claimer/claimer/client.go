package claimer

import (
	"context"
	"errors"
	"fmt"

	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/app/params"
	bep3types "github.com/kava-labs/kava/x/bep3/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type KavaClient struct {
	encodingConfig params.EncodingConfig
	cdc            *codec.LegacyAmino
	http           *rpcclient.HTTP
}

func NewKavaClient(rpcAddr string, logger log.Logger) (*KavaClient, error) {
	http, err := rpcclient.New(rpcAddr, "/websocket")
	if err != nil {
		return nil, err
	}
	http.Logger = logger

	encodingConfig := app.MakeEncodingConfig()

	return &KavaClient{
		cdc:            encodingConfig.Amino,
		encodingConfig: encodingConfig,
		http:           http,
	}, nil
}

func (c *KavaClient) GetChainID(ctx context.Context) (string, error) {
	result, err := c.http.Status(ctx)
	if err != nil {
		return "", err
	}
	return result.NodeInfo.Network, nil
}

func (c *KavaClient) GetAccount(ctx context.Context, addr sdk.AccAddress) (acc authtypes.AccountI, err error) {
	params := authtypes.QueryAccountRequest{Address: addr.String()}
	bz, err := c.cdc.MarshalJSON(params)

	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("custom/acc/account/%s", addr.String())

	result, err := c.ABCIQuery(ctx, path, bz)
	if err != nil {
		return nil, err
	}

	err = c.cdc.UnmarshalJSON(result, &acc)
	if err != nil {
		return nil, err
	}

	return acc, err
}

func (c *KavaClient) GetSwapByID(ctx context.Context, swapID []byte) (bep3types.AtomicSwap, error) {
	params := bep3types.NewQueryAtomicSwapByID(swapID)
	bz, err := c.cdc.MarshalJSON(params)
	if err != nil {
		return bep3types.AtomicSwap{}, err
	}

	result, err := c.ABCIQuery(ctx, "custom/bep3/swap", bz)
	if err != nil {
		return bep3types.AtomicSwap{}, err
	}

	var swap bep3types.AtomicSwap
	err = c.cdc.UnmarshalJSON(result, &swap)
	if err != nil {
		return bep3types.AtomicSwap{}, err
	}
	return swap, nil
}

func (c *KavaClient) ABCIQuery(ctx context.Context, path string, data tmbytes.HexBytes) ([]byte, error) {
	result, err := c.http.ABCIQuery(ctx, path, data)
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

func (c *KavaClient) BroadcastTxSync(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	return c.http.BroadcastTxSync(ctx, tx)
}

func (c *KavaClient) GetTxConfirmation(ctx context.Context, txHash []byte) (*ctypes.ResultTx, error) {
	return c.http.Tx(ctx, txHash, false)
}
