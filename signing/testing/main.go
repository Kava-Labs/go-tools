package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/kava-labs/go-tools/signing"
	"github.com/kava-labs/kava/app"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

const (
	inflightLimit = 100
	grpcUrl       = "localhost:9090"
	mnemonic      = "season bone lucky dog depth pond royal decide unknown device fruit inch clock trap relief horse morning taxi bird session throw skull avocado private"
	toAddress     = "kava1mq9qxlhze029lm0frzw2xr6hem8c3k9ts54w0w"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Set up colored logging
	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}).With().Timestamp().Logger()

	app.SetSDKConfig()
	encodingConfig := app.MakeEncodingConfig()

	// Set up the gRPC connection
	conn, err := grpc.Dial(grpcUrl, grpc.WithInsecure())
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to gRPC server")
	}
	defer conn.Close()

	// Initialize Tendermint service client and retrieves node information
	tmClient := tmservice.NewServiceClient(conn)
	nodeInfoResponse, err := tmClient.GetNodeInfo(ctx, &tmservice.GetNodeInfoRequest{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to get node info")
	}

	// Initialize clients for sending transactions and querying authentication-related data
	txClient := txtypes.NewServiceClient(conn)
	authClient := authtypes.NewQueryClient(conn)

	// Derives a private key from the provided mnemonic using the HD path
	// Converts the private key to a `secp256k1.PrivKey` and generates the account address
	hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, 0)
	privKeyBytes, err := hd.Secp256k1.Derive()(mnemonic, "", hdPath.String())
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to derive private key")
	}
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}
	accAddr := sdk.AccAddress(privKey.PubKey().Address())

	// Initialize signer to handle signing and sending transactions
	signer, err := signing.NewSigner(
		nodeInfoResponse.DefaultNodeInfo.Network,
		signing.EncodingConfigAdapter{EncodingConfig: encodingConfig},
		authClient,
		txClient,
		privKey,
		inflightLimit,
		logger,
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to initialize signer")
	}
	// Create channel for sending message requests to the signer and starts the signer
	requests := make(chan signing.MsgRequest)
	responses, err := signer.Run(ctx, requests)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to start signer")
	}

	// Set up a signal handler to gracefully shutdown upon receiving termination signal
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopCh
		logger.Info().Msg("Received termination signal. Exiting...")
		close(requests)
		os.Exit(0)
	}()

	// Process responses from signer
	go func() {
		for {
			response := <-responses
			if response.Err != nil {
				logger.Error().
					Uint32("code", response.Result.Code).
					Err(response.Err).
					Msg("Transaction failed")
				continue
			}
			logger.Info().
				Uint32("code", response.Result.Code).
				Str("hash", response.Result.TxHash).
				Msg("Transaction successful")
		}
	}()

	// Create and send messages
	toAddr, err := sdk.AccAddressFromBech32(toAddress)
	if err != nil {
		logger.Fatal().Err(err).Msg("Invalid destination address")
	}
	msg := banktypes.NewMsgSend(
		accAddr,
		toAddr,
		sdk.NewCoins(
			sdk.NewCoin("ukava", sdk.NewInt(1)),
		),
	)

	for i := 0; i < 1000; i++ {
		requests <- signing.MsgRequest{
			Msgs:     []sdk.Msg{msg},
			GasLimit: 200000,
			FeeAmount: sdk.NewCoins(
				sdk.NewCoin("ukava", sdk.NewInt(1000)),
			),
		}

		logger.Info().Int("msg", i+1).Msg("Sent message")
	}

	// Block indefinitely to keep the program running.
	select {}
}
