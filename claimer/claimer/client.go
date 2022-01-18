package claimer

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	tmbytes "github.com/tendermint/tendermint/libs/bytes"

	"github.com/kava-labs/kava/app/params"
	bep3types "github.com/kava-labs/kava/x/bep3/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

//go:generate mockgen -destination mock/kava_client.go -package mock . KavaChainClient

type KavaChainClient interface {
	GetTxConfirmation(txHash string) (*sdk.TxResponse, error)
	GetAccount(address sdk.AccAddress) (authtypes.AccountI, error)
	GetChainID() (string, error)
	BroadcastTx(tx txtypes.BroadcastTxRequest) (*txtypes.BroadcastTxResponse, error)
	GetSwapByID(id tmbytes.HexBytes) (bep3types.AtomicSwapResponse, error)
	GetEncodingCoding() params.EncodingConfig
}

var _ KavaChainClient = grpcKavaClient{}

type grpcKavaClient struct {
	encodingConfig params.EncodingConfig
	GrpcClientConn *grpc.ClientConn
	Auth           authtypes.QueryClient
	Bep3           bep3types.QueryClient
	Tx             txtypes.ServiceClient
	Tm             tmservice.ServiceClient
}

func NewGrpcKavaClient(target string, encodingConfig params.EncodingConfig) grpcKavaClient {
	grpcUrl, err := url.Parse(target)
	if err != nil {
		log.Fatal(err)
	}

	var secureOpt grpc.DialOption
	switch grpcUrl.Scheme {
	case "http":
		secureOpt = grpc.WithInsecure()
	case "https":
		creds := credentials.NewTLS(&tls.Config{})
		secureOpt = grpc.WithTransportCredentials(creds)
	default:
		log.Fatalf("unknown rpc url scheme %s\n", grpcUrl.Scheme)
	}

	grpcConn, err := grpc.Dial(grpcUrl.Host, secureOpt)
	if err != nil {
		panic(err)
	}

	return grpcKavaClient{
		encodingConfig: encodingConfig,
		GrpcClientConn: grpcConn,
		Auth:           authtypes.NewQueryClient(grpcConn),
		Bep3:           bep3types.NewQueryClient(grpcConn),
		Tm:             tmservice.NewServiceClient(grpcConn),
		Tx:             txtypes.NewServiceClient(grpcConn),
	}
}

func (kc grpcKavaClient) GetAccount(address sdk.AccAddress) (authtypes.AccountI, error) {
	res, err := kc.Auth.Account(context.Background(), &authtypes.QueryAccountRequest{
		Address: address.String(),
	})
	if err != nil {
		return nil, err
	}

	var acc authtypes.AccountI
	err = kc.encodingConfig.Marshaler.UnpackAny(res.Account, &acc)
	if err != nil {
		return nil, err
	}

	return acc, nil
}

func (kc grpcKavaClient) GetTxConfirmation(txHash string) (*sdk.TxResponse, error) {
	res, err := kc.Tx.GetTx(context.Background(), &txtypes.GetTxRequest{
		Hash: txHash,
	})
	if err != nil {
		return nil, err
	}

	return res.TxResponse, nil
}

func (kc grpcKavaClient) BroadcastTx(tx txtypes.BroadcastTxRequest) (*txtypes.BroadcastTxResponse, error) {
	res, err := kc.Tx.BroadcastTx(context.Background(), &tx)
	if err != nil {
		return res, err
	}

	if res.TxResponse.Code != 0 { // tx failed to be submitted to the mempool
		return res, fmt.Errorf("transaction failed to get into mempool: %s", res.TxResponse.RawLog) // TODO should return a named error
	}
	return res, nil
}

func (kc grpcKavaClient) GetChainID() (string, error) {
	latestBlock, err := kc.Tm.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		return "", err
	}

	return latestBlock.Block.Header.ChainID, nil
}

func (kc grpcKavaClient) GetSwapByID(id tmbytes.HexBytes) (bep3types.AtomicSwapResponse, error) {
	strID := strings.ToLower(hex.EncodeToString(id))
	res, err := kc.Bep3.AtomicSwap(context.Background(), &bep3types.QueryAtomicSwapRequest{
		SwapId: strID,
	})
	if err != nil {
		return bep3types.AtomicSwapResponse{}, err
	}

	return res.AtomicSwap, nil
}

func (kc grpcKavaClient) GetEncodingCoding() params.EncodingConfig {
	return kc.encodingConfig
}
