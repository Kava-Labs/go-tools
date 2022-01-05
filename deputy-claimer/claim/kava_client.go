package claim

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
	"github.com/cosmos/cosmos-sdk/types/query"
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
	GetOpenOutgoingSwaps() ([]bep3types.AtomicSwapResponse, error)
	GetRandomNumberFromSwap(id []byte) ([]byte, error)
	GetTxConfirmation(txHash []byte) (*sdk.TxResponse, error)
	GetAccount(address sdk.AccAddress) (authtypes.AccountI, error)
	GetChainID() (string, error)
	BroadcastTx(tx txtypes.BroadcastTxRequest) error
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

func (kc grpcKavaClient) GetOpenOutgoingSwaps() ([]bep3types.AtomicSwapResponse, error) {
	res, err := kc.Bep3.AtomicSwaps(context.Background(), &bep3types.QueryAtomicSwapsRequest{
		Status:    bep3types.SWAP_STATUS_OPEN,
		Direction: bep3types.SWAP_DIRECTION_OUTGOING,
		Pagination: &query.PageRequest{
			Limit: 1000,
		},
	})

	if err != nil {
		return nil, err
	}

	return res.AtomicSwaps, nil
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

func (kc grpcKavaClient) GetTxConfirmation(txHash []byte) (*sdk.TxResponse, error) {
	txHashStr := strings.ToLower(hex.EncodeToString(txHash))
	res, err := kc.Tx.GetTx(context.Background(), &txtypes.GetTxRequest{
		Hash: txHashStr,
	})
	if err != nil {
		return nil, err
	}

	return res.TxResponse, nil
}

func (kc grpcKavaClient) BroadcastTx(tx txtypes.BroadcastTxRequest) error {
	res, err := kc.Tx.BroadcastTx(context.Background(), &tx)
	if err != nil {
		return err
	}

	if res.TxResponse.Code != 0 { // tx failed to be submitted to the mempool
		return fmt.Errorf("transaction failed to get into mempool: %s", res.TxResponse.RawLog) // TODO should return a named error
	}
	return nil
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

func (kc grpcKavaClient) GetRandomNumberFromSwap(id []byte) ([]byte, error) {
	strID := strings.ToLower(hex.EncodeToString(id))
	query := fmt.Sprintf(`claim_atomic_swap.atomic_swap_id='%s'`, strID) // must be lowercase hex for querying to work

	// Event format is "{eventType}.{eventAttribute}={value}"
	// https://github.com/cosmos/cosmos-sdk/blob/9fd866e3820b3510010ae172b682d71594cd8c14/x/auth/tx/service.go#L43
	res, err := kc.Tx.GetTxsEvent(context.Background(), &txtypes.GetTxsEventRequest{
		Events: []string{
			query,
		},
	})

	if err != nil {
		return nil, err
	}

	if len(res.Txs) == 0 {
		return nil, fmt.Errorf("no claim txs found")
	}

	parsedTx, err := kc.encodingConfig.TxConfig.TxDecoder()(res.TxResponses[0].Tx.Value)
	if err != nil {
		return nil, fmt.Errorf("decode tx error: %w", err)
	}

	for _, msg := range parsedTx.GetMsgs() {
		switch realMsg := msg.(type) {
		case *bep3types.MsgClaimAtomicSwap:
			return realMsg.RandomNumber, nil
		}
	}

	return nil, fmt.Errorf("no claim tx found")
}
