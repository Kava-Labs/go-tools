package main

import (
	"encoding/hex"
	"fmt"

	butil "github.com/binance-chain/bep3-deputy/util"
	brpc "github.com/binance-chain/go-sdk/client/rpc"
	bkeys "github.com/binance-chain/go-sdk/keys"
	ec "github.com/ethereum/go-ethereum/common"

	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-sdk/kava"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/pool"
)

// TODO:
// Check blockchain for the expected swap ID
// Once the swap is available, submit the claim request to a worker pool of claimers
// Request goes to most available worker
// Claimer must wait 6 seconds after sending claim to be assigned next claim
// Check that the swap is still available (if expired/completed, remove request from worker pool)
// Use go-sdk's Kava codec for sending claims (RPC or REST?)
// Check tx result - if tx was unsuccessful, resubmit the claim to the worker pool

const (
	mnemonic       = "suggest wheel pool person mass gorilla day bachelor invite walk upset want clown firm pen wet laundry exact guard goat stumble vocal dial similar"
	mnemonicAddr   = "kava1gy5gng82jnes05qq5rsa5wxktdpx9hfz2v2ftf"
	rpcAddr        = "tcp://kava3.data.kava.io"
	networkMainnet = 2

	localRpcAddr = "tcp://localhost:26657"
	networkLocal = 0

	bncMnemonic    = "city neither hub forum chalk treat recall cupboard play emotion elephant matter narrow noodle audit infant priority cloth plug card knee scorpion broken meadow"
	bncAddrMainnet = "bnb165v0088yden5849y0hsr4p8v2k03raa3jdc892"
	bncRpcAddr     = "http://dataseed1.binance.org:80" // THIS IS MAINNET
	bncNetwork     = 1
)

func main() {
	// Set up Kava client
	config := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(config)
	cdc := kava.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, mnemonic, kava.Bip44CoinType, localRpcAddr, networkLocal)

	// Set up Binance client
	bncClient := brpc.NewRPCClient(bncRpcAddr, bncNetwork)
	keyManager, err := bkeys.NewMnemonicKeyManager(bncMnemonic)
	if err != nil {
		panic(err)
	}
	bncClient.SetKeyManager(keyManager)
	bncClient.SetLogger(butil.SdkLogger)

	claimer := claimer.NewClaimer(kavaClient, bncClient)
	_ = claimer

	initPool()

}

func initPool() {
	collector := pool.StartDispatcher(5) // start up worker pool

	for i, job := range pool.CreateJobs(100) {
		collector.Work <- pool.Work{Job: job, ID: i}
	}
}

func testBinance(claimer claimer.Claimer) {
	rawBinanceSwapID := "d696e04dca49a453b6ad38a4da7c1f457de7959a856513452eb7bf958d7131ca"
	rawHash := ec.HexToHash(rawBinanceSwapID)
	fmt.Println(claimer.IsClaimableBinance(rawHash))
}

func testKava(claimer claimer.Claimer) {
	rawSwapID := "91f273315e658c27752014c8011a66f8876b8b64fa42570ea793eed181859982"
	swapID, err := hex.DecodeString(rawSwapID)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(claimer.IsClaimableKava(swapID))
}
