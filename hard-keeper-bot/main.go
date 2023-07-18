package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/go-tools/signing"
	"github.com/kava-labs/kava/app"
	"github.com/rs/zerolog"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	app.SetSDKConfig()
	encodingConfig := app.MakeEncodingConfig()
	logger := zerolog.New(os.Stderr)

	config, err := LoadConfig(&EnvLoader{})
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	grpcUrl, err := url.Parse(config.KavaGrpcUrl)
	if err != nil {
		logger.Fatal().Err(err).Send()
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

	http, err := rpchttpclient.New(config.KavaRpcUrl, "/websocket")
	if err != nil {
		logger.Fatal().Err(err).Send()
	}
	liquidationClient := NewRpcLiquidationClient(http, encodingConfig.Amino)

	conn, err := grpc.Dial(grpcUrl.Host, secureOpt)
	if err != nil {
		logger.Fatal().Err(err).Send()
	}
	defer conn.Close()

	tmClient := tmservice.NewServiceClient(conn)
	nodeInfoResponse, err := tmClient.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	if err != nil {
		logger.Fatal().Err(err).Send()
	}

	txClient := txtypes.NewServiceClient(conn)
	authClient := authtypes.NewQueryClient(conn)

	hdPath := hd.CreateHDPath(app.Bip44CoinType, 0, 0)
	privKeyBytes, err := hd.Secp256k1.Derive()(config.KavaSignerMnemonic, "", hdPath.String())
	if err != nil {
		panic(err)
	}
	privKey := &secp256k1.PrivKey{Key: privKeyBytes}

	signer := signing.NewSigner(
		nodeInfoResponse.DefaultNodeInfo.Network,
		encodingConfig,
		authClient,
		txClient,
		privKey,
		10,
		logger,
	)

	// channels to communicate with signer
	requests := make(chan signing.MsgRequest)

	// signer starts it's own go routines and returns
	responses, err := signer.Run(requests)
	if err != nil {
		logger.Fatal().Err(err).Send()
		os.Exit(1)
	}

	// log responses, if responses are not read, requests will block
	go func() {
		for {
			// response is not returned until the msg is committed to a block
			response := <-responses

			// error will be set if response is not Code 0 (success) or Code 19 (already in mempool)
			if response.Err != nil {
				fmt.Printf("response code: %d error %s\n", response.Result.Code, response.Err)
				continue
			}

			// code and result are from broadcast, not deliver tx
			// it is up to the caller/requester to check the deliver tx code and deal with failure
			fmt.Printf("response code: %d, hash %s\n", response.Result.Code, response.Result.TxHash)
		}
	}()

	for {
		// fetch asset and position data using client
		data, err := GetPositionData(liquidationClient)
		if err != nil {
			log.Println(err)
			continue
		}

		// calculate borrowers to liquidate from asset and position data
		borrowersToLiquidate := GetBorrowersToLiquidate(data)
		fmt.Printf("%d borrowers to liquidate\n", len(borrowersToLiquidate))

		// create liquidation msgs
		msgs := CreateLiquidationMsgs(config.KavaKeeperAddress, borrowersToLiquidate)

		// create liquidation transactions
		for _, msg := range msgs {
			fmt.Printf("sending liquidation for %s\n", msg.Borrower)

			requests <- signing.MsgRequest{
				Msgs:      []sdk.Msg{&msg},
				GasLimit:  1000000,
				FeeAmount: sdk.Coins{sdk.Coin{Denom: "ukava", Amount: sdk.NewInt(50000)}},
				Memo:      "",
			}
		}

		// wait for next interval
		time.Sleep(config.KavaLiquidationInterval)
	}
}
