package claimer

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/log"
	rpcclient "github.com/tendermint/tendermint/rpc/client/http"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	app "github.com/kava-labs/kava/app"
	bep3 "github.com/kava-labs/kava/x/bep3/types"
)

type KavaClient struct {
	http *rpcclient.HTTP
	cdc  *codec.LegacyAmino
	ctx  client.Context
}

func NewKavaClient(cdc *codec.LegacyAmino, rpcAddr string, logger log.Logger) (*KavaClient, error) {
	http, err := rpcclient.New(rpcAddr, "/websocket")
	if err != nil {
		return nil, err
	}
	http.Logger = logger

	encodingConfig := app.MakeEncodingConfig()
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithNodeURI(rpcAddr).
		WithClient(http).
		WithAccountRetriever(authtypes.AccountRetriever{})

	return &KavaClient{
		cdc:  cdc,
		http: http,
		ctx:  clientCtx,
	}, nil
}

// GetSwapByID gets an atomic swap on Kava by ID
func (kc *KavaClient) GetSwapByID(ctx context.Context, swapID tmbytes.HexBytes) (swap bep3.AtomicSwap, err error) {
	params := bep3.NewQueryAtomicSwapByID(swapID)
	bz, err := kc.cdc.MarshalJSON(params)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}

	path := "custom/bep3/swap"

	result, err := kc.ABCIQuery(ctx, path, bz)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}

	err = kc.cdc.UnmarshalJSON(result, &swap)
	if err != nil {
		return bep3.AtomicSwap{}, err
	}
	return swap, nil
}

// GetAccount gets the account associated with an address on Kava
func (kc *KavaClient) GetAccount(ctx context.Context, addr sdk.AccAddress) (acc authtypes.BaseAccount, err error) {
	params := authtypes.QueryAccountRequest{Address: addr.String()}
	bz, err := kc.cdc.MarshalJSON(params)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	path := fmt.Sprintf("custom/auth/account/%s", addr.String())

	result, err := kc.ABCIQuery(ctx, path, bz)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	err = kc.cdc.UnmarshalJSON(result, &acc)
	if err != nil {
		return authtypes.BaseAccount{}, err
	}

	return acc, err
}

func (kc *KavaClient) GetChainID(ctx context.Context) (string, error) {
	result, err := kc.http.Status(ctx)
	if err != nil {
		return "", err
	}
	return result.NodeInfo.Network, nil
}

// ABCIQuery sends a query to Kava
func (kc *KavaClient) ABCIQuery(ctx context.Context, path string, data tmbytes.HexBytes) ([]byte, error) {
	result, err := kc.http.ABCIQuery(ctx, path, data)
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

func (c *KavaClient) GetTxConfirmation(ctx context.Context, txHash []byte) (*ctypes.ResultTx, error) {
	return c.http.Tx(ctx, txHash, false)
}
