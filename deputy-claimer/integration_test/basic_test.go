// +build integration

package main

import (
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

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
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
	timestamp := time.Now().Unix() - 10*60 - 1 // set the timestamp to be in the past
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

	_, err = kavaClient.Broadcast(createMsg, client.Sync)
	require.NoError(t, err)

	// send corresponding htlt on bnb
	_, err = bnbClient.HTLT(
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

	// claim bnb htlt
	time.Sleep(3 * time.Second)
	bnbID := msg.CalculateSwapID(rndHash, common.BnbDeputyAddr, common.KavaUserAddrs[0].String())
	_, err = bnbClient.ClaimHTLT(bnbID, rndNum, bnbRpc.Sync)
	require.NoError(t, err)

	// run thing
	time.Sleep(5 * time.Second)
	err = claim.RunKava("http://localhost:1317", "tcp://localhost:26657", "tcp://localhost:26658", "bnb1uky3me9ggqypmrsvxk7ur6hqkzq7zmv4ed4ng7", common.KavaUserMnemonics[:2])
	require.NoError(t, err)

	// check kava claims were claimed
	kavaSwapID := bep3.CalculateSwapID(rndHash, kavaKeyM.GetAddr(), common.BnbDeputyAddr.String())
	_, err = kavaClient.GetSwapByID(kavaSwapID)
	require.NoError(t, err)
}
