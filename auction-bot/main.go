package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	kava "github.com/kava-labs/kava/app"
	"github.com/tendermint/tendermint/libs/log"
)

const (
	kavaGrpcUrlEnvKey = "KAVA_GRPC_URL"
	enableTlsEnvKey   = "GRPC_TLS"
	mnemonicEnvKey    = "KEEPER_MNEMONIC"
	profitMarginKey   = "BID_MARGIN"
	bidIntervalKey    = "BID_INTERVAL"
)

func main() {
	// create base logger
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	//
	// bootstrap kava chain config
	//
	// sets a global cosmos sdk for bech32 prefix
	//
	// required before loading config
	//
	kavaConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(kavaConfig)
	kava.SetBip44CoinType(kavaConfig)
	kavaConfig.Seal()

	//
	// Load config
	//
	// if config is not valid, exit with fatal error
	//
	config, err := LoadConfig(&EnvLoader{})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.With(
		"grpcUrl", config.KavaGrpcUrl,
		"bidInterval", config.KavaBidInterval.String(),
		"profitMargin", config.ProfitMargin.String(),
	).Info("config loaded")

	//
	// create codec for messages
	//
	encodingConfig := kava.MakeEncodingConfig()

	//
	// create rpc client for fetching data
	// required for bidding
	//
	logger.Info("creating grpc client")

	grpcClient := NewGrpcClient(config.KavaGrpcUrl, config.EnableTls, encodingConfig.Marshaler)
	defer grpcClient.GrpcClientConn.Close()

	//
	// client for broadcasting txs
	//
	params := *hd.NewFundraiserParams(0, 459, 0)
	hdPath := params.String()

	privKeyBytes, err := hd.Secp256k1.Derive()(config.KavaKeeperMnemonic, "", hdPath)
	if err != nil {
		logger.Error("failed to derive key")
		logger.Error(err.Error())
		os.Exit(1)
	}
	// wrap with cosmos secp256k1 private key struct
	privKey := secp256k1.PrivKey{Key: privKeyBytes}
	logger.Info(fmt.Sprintf("signing address: %s", sdk.AccAddress(privKey.PubKey().Address()).String()))

	signer := NewSigner(encodingConfig, grpcClient, privKey, 10)

	// channels to communicate with signer
	requests := make(chan MsgRequest)

	// signer starts it's own go routines and returns
	responses, err := signer.Run(requests)
	if err != nil {
		logger.Error("failed to start signer")
		logger.Error(err.Error())
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
		data, err := GetAuctionData(grpcClient, encodingConfig.Marshaler)
		if err != nil {
			fmt.Printf("error fetching prices...")
			continue
		}

		latestHeight, err := grpcClient.LatestHeight()
		if err != nil {
			continue
		}

		logger.Info(fmt.Sprintf("latest height: %d", latestHeight))
		logger.Info(fmt.Sprintf("checking %d auctions", len(data.Auctions)))

		auctionBids := GetBids(data, sdk.AccAddress(privKey.PubKey().Address()), config.ProfitMargin)

		msgs := CreateBidMsgs(sdk.AccAddress(privKey.PubKey().Address()), auctionBids)

		logger.Info(fmt.Sprintf("creating %d bids", len(msgs)))

		for _, msg := range msgs {
			requests <- MsgRequest{
				Msgs: []sdk.Msg{&msg},
				Fee:  sdk.Coins{sdk.Coin{Denom: "ukava", Amount: sdk.NewInt(15000)}},
				Memo: "",
			}
		}

		// wait for next interval
		time.Sleep(config.KavaBidInterval)
	}
}
