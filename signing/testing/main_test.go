//go:build integration

package main_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os"
	"testing"
	"time"

	"github.com/kava-labs/go-tools/signing"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/kava-labs/kava/app"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

const (
	inflightLimit = 100
	grpcUrl       = "localhost:9090"
	mnemonic      = "season bone lucky dog depth pond royal decide unknown device fruit inch clock trap relief horse morning taxi bird session throw skull avocado private"
	toAddress     = "kava1mq9qxlhze029lm0frzw2xr6hem8c3k9ts54w0w"

	testNumber = 1000
	amountSend = 321
)

func TestSigningTransactions(t *testing.T) {
	app.SetSDKConfig()
	encodingConfig := app.MakeEncodingConfig()

	conn, err := grpc.Dial(grpcUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to grpc server: %s", err)
	}
	defer conn.Close()

	tmClient := tmservice.NewServiceClient(conn)
	nodeInfoResponse, err := tmClient.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	if err != nil {
		log.Fatal(err)
	}

	txClient := txtypes.NewServiceClient(conn)
	authClient := authtypes.NewQueryClient(conn)

	hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, 0)
	privKeyBytes, err := hd.Secp256k1.Derive()(mnemonic, "", hdPath.String())
	if err != nil {
		t.Fatalf("failed to derive private key: %s", err)
	}

	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	accAddr := sdk.AccAddress(privKey.PubKey().Address())

	logger := zerolog.New(os.Stderr)

	signer := signing.NewSigner(
		nodeInfoResponse.DefaultNodeInfo.Network,
		signing.EncodingConfigAdapter{EncodingConfig: encodingConfig},
		authClient,
		txClient,
		privKey,
		inflightLimit,
		logger,
	)
	requests := make(chan signing.MsgRequest)
	responses, err := signer.Run(requests)
	if err != nil {
		t.Fatalf("failed to start signer: %s", err)
	}

	group := errgroup.Group{}

	group.Go(func() error {
		counter := 0

		for {
			select {
			case response := <-responses:
				counter++
				if counter == testNumber {
					log.Printf("received all responses")

					return nil
				}

				if response.Err != nil {
					t.Errorf("response error: %s", response.Err)

					return response.Err
				}

			// if we cannot receive all responses in 10 seconds, we fail the test
			case <-time.After(10 * time.Second):
				t.Fatalf("timeout waiting for responses")
			}
		}
	})

	toAddr, err := sdk.AccAddressFromBech32(toAddress)
	if err != nil {
		t.Fatalf("failed to parse address: %s", err)
	}

	grpcClient := banktypes.NewQueryClient(conn)
	addressBefore, err := getAddress(context.Background(), grpcClient, toAddress)
	require.NoError(t, err)

	log.Printf("address before: %v", addressBefore.Amount.Int64())

	msg := banktypes.NewMsgSend(accAddr, toAddr, sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(amountSend))))

	for i := 0; i < testNumber; i++ {
		requests <- signing.MsgRequest{
			Msgs:      []sdk.Msg{msg},
			GasLimit:  200000,
			FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(1000))),
		}
	}

	log.Printf("all transactions sent")

	if err := group.Wait(); err != nil {
		t.Fatalf("failed to sign transactions: %s", err)
	}

	log.Printf("all transactions sent, waiting for block generation")

	// we need to wait for a block to be generated after last transactions
	time.Sleep(10 * time.Second)

	address, err := getAddress(context.Background(), grpcClient, toAddress)
	require.NoError(t, err)
	log.Printf("address after: %v", address.Amount.Int64())
	require.Equal(t, addressBefore.Amount.Int64()+int64(amountSend)*testNumber, address.Amount.Int64())

	log.Printf("all transactions sent, waiting for block generation")
}

// getAddress gets the balance of the given address
func getAddress(
	ctx context.Context,
	//grpcClient *kavagrpc.KavaGrpcClient,
	grpcClient banktypes.QueryClient,
	address string,
) (*sdk.Coin, error) {
	// Get the current pool state
	res, err := grpcClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   "ukava",
	})
	if err != nil {
		return nil, fmt.Errorf("query bank: %v", err)
	}

	return res.Balance, nil
}
