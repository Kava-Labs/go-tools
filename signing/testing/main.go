package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kava-labs/go-tools/signing"
	"github.com/rs/zerolog"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/kava-labs/kava/app"
	"google.golang.org/grpc"
)

const (
	inflightLimit = 100
	grpcUrl       = "localhost:9090"
	mnemonic      = "season bone lucky dog depth pond royal decide unknown device fruit inch clock trap relief horse morning taxi bird session throw skull avocado private"
	toAddress     = "kava1mq9qxlhze029lm0frzw2xr6hem8c3k9ts54w0w"
)

func main() {
	app.SetSDKConfig()
	encodingConfig := app.MakeEncodingConfig()

	conn, err := grpc.Dial(grpcUrl, grpc.WithInsecure())
	if err != nil {
		panic(err)
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
		panic(err)
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
		fmt.Println("failed to start signer")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	go func() {
		for {
			response := <-responses
			if response.Err != nil {
				fmt.Printf("response code: %d error %s\n", response.Result.Code, response.Err)
				continue
			}
			fmt.Printf("response code: %d, hash %s\n", response.Result.Code, response.Result.TxHash)
		}
	}()

	toAddr, err := sdk.AccAddressFromBech32(toAddress)
	if err != nil {
		panic(err)
	}
	msg := banktypes.NewMsgSend(accAddr, toAddr, sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(1))))

	for i := 0; i < 1000; i++ {
		requests <- signing.MsgRequest{
			Msgs:      []sdk.Msg{msg},
			GasLimit:  200000,
			FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(1000))),
		}

		fmt.Printf("sent msg %d\n", i+1)
	}

	for {
	}
}
