// notabuildcmd+build integration
package main

import (
	"fmt"
	"testing"
	"time"

	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"

	bnbKeys "github.com/binance-chain/go-sdk/keys"
	"github.com/binance-chain/go-sdk/types/msg"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/integration_test/common"
)

func TestBasic(t *testing.T) {
	// setup clients
	cdc := kava.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, common.KavaUserMnemonics[0], kava.Bip44CoinType, common.KavaNodeURL, client.LocalNetwork)
	kavaKeyM, err := kavaKeys.NewMnemonicKeyManager(common.KavaUserMnemonics[0], kava.Bip44CoinType)
	require.NoError(t, err)
	bnbKeyM, err := bnbKeys.NewMnemonicKeyManager(common.BnbDeputyMnemonic)
	require.NoError(t, err)
	bnbClient := bnbRpc.NewRPCClient(common.BnbNodeURL, types.ProdNetwork)
	bnbClient.SetKeyManager(bnbKeyM)

	// send htlt on kva
	rndNum, err := bep3.GenerateSecureRandomNumber()
	require.NoError(t, err)
	timestamp := time.Now().Unix()
	rndHash := bep3.CalculateRandomHash(rndNum, timestamp)
	createMsg := bep3.NewMsgCreateAtomicSwap(
		kavaKeyM.GetAddr(),              // sender
		common.KavaDeputyAddr,           // recipient
		common.BnbUserAddrs[0].String(), // recipient other chain
		common.BnbDeputyAddr.String(),   // sender other chain
		rndHash,
		timestamp,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 100_000_000)),
		250,
	)
	require.NoError(t, createMsg.ValidateBasic())

	fmt.Println(createMsg.GetSigners())
	fmt.Println(kavaKeyM.GetAddr())

	resK, err := kavaClient.Broadcast(createMsg, client.Sync)
	require.NoError(t, err)
	fmt.Printf("htlt res: %+v\n", resK)

	// send corresponding htlt on bnb
	resB, err := bnbClient.HTLT(
		common.BnbUserAddrs[0],           // recipient
		common.KavaDeputyAddr.String(),   // recipient other chain
		common.KavaUserAddrs[0].String(), // other chain sender
		rndHash,
		timestamp,
		types.Coins{{Denom: "BNB", Amount: 100_000_000}}, //{Denom: "BNB", Amount: 100_000_000}},
		"",  // expected income
		360, // heightspan
		true,
		bnbRpc.Sync,
	)
	require.NoError(t, err)
	fmt.Printf("htlt res: %+v\n", resB)

	// claim bnb htlt
	time.Sleep(3 * time.Second)
	bnbID := msg.CalculateSwapID(rndHash, common.BnbDeputyAddr, common.KavaUserAddrs[0].String())
	resB, err = bnbClient.ClaimHTLT(bnbID, rndNum, bnbRpc.Sync)
	require.NoError(t, err)
	fmt.Printf("claim res: %+v\n", resB)

	// run thing
	time.Sleep(5 * time.Second)
	err = RunKava("http://localhost:1317", "tcp://localhost:26658", "bnb1uky3me9ggqypmrsvxk7ur6hqkzq7zmv4ed4ng7")
	require.NoError(t, err)

	// check kava claims were claimed
	// TODO
}
