// +build integration

package main

import (
	"context"
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

func TestClaimKava(t *testing.T) {
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

	_, err = kavaClient.Broadcast(createMsg, client.Commit)
	require.NoError(t, err)

	// send another htlt on kava
	rndNum2, err := bep3.GenerateSecureRandomNumber()
	require.NoError(t, err)
	rndHash2 := bep3.CalculateRandomHash(rndNum2, timestamp)
	createMsg2 := bep3.NewMsgCreateAtomicSwap(
		kavaKeyM.GetAddr(),              // sender
		common.KavaDeputyAddr,           // recipient
		common.BnbUserAddrs[0].String(), // recipient other chain
		common.BnbDeputyAddr.String(),   // sender other chain
		rndHash2,
		timestamp,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 100_000_000)),
		250,
	)
	require.NoError(t, createMsg2.ValidateBasic())

	_, err = kavaClient.Broadcast(createMsg2, client.Commit)
	require.NoError(t, err)

	// send corresponding htlts on bnb
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
		bnbRpc.Commit,
	)
	require.NoError(t, err)

	_, err = bnbClient.HTLT(
		common.BnbUserAddrs[0],           // recipient
		common.KavaDeputyAddr.String(),   // recipient other chain
		common.KavaUserAddrs[0].String(), // other chain sender
		rndHash2,
		timestamp,
		types.Coins{{Denom: "BNB", Amount: 100_000_000}}, //{Denom: "BNB", Amount: 100_000_000}},
		"",  // expected income
		360, // heightspan
		true,
		bnbRpc.Commit,
	)
	require.NoError(t, err)

	// claim first bnb htlt
	time.Sleep(3 * time.Second)
	bnbID := msg.CalculateSwapID(rndHash, common.BnbDeputyAddr, common.KavaUserAddrs[0].String())
	_, err = bnbClient.ClaimHTLT(bnbID, rndNum, bnbRpc.Sync)
	require.NoError(t, err)

	// run
	time.Sleep(5 * time.Second) // TODO replace with wait func
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewKavaClaimer("http://localhost:1317", "tcp://localhost:26657", "tcp://localhost:26658", "bnb1uky3me9ggqypmrsvxk7ur6hqkzq7zmv4ed4ng7", common.KavaUserMnemonics[:2]).Run(ctx)
	defer shutdownClaimer()
	time.Sleep(8 * time.Second)

	// check the first kava swap was claimed
	kavaSwapID := bep3.CalculateSwapID(rndHash, kavaKeyM.GetAddr(), common.BnbDeputyAddr.String())
	s, err := kavaClient.GetSwapByID(kavaSwapID)
	require.NoError(t, err)
	require.Equal(t, bep3.Completed, s.Status)
}

func TestClaimBnb(t *testing.T) {
	// setup clients
	cdc := kava.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, common.KavaDeputyMnemonic, kava.Bip44CoinType, common.KavaNodeURL, client.LocalNetwork)
	bnbKeyM, err := bnbKeys.NewMnemonicKeyManager(common.BnbUserMnemonics[0])
	require.NoError(t, err)
	bnbClient := bnbRpc.NewRPCClient(common.BnbNodeURL, types.ProdNetwork)
	bnbClient.SetKeyManager(bnbKeyM)

	// Create a swap on each chain
	rndNum, err := bep3.GenerateSecureRandomNumber()
	require.NoError(t, err)
	timestamp := time.Now().Unix() - 10*60 - 1 // set the timestamp to be in the past
	rndHash := bep3.CalculateRandomHash(rndNum, timestamp)
	_, err = bnbClient.HTLT(
		common.BnbDeputyAddr,             // recipient
		common.KavaUserAddrs[0].String(), // recipient other chain
		common.KavaDeputyAddr.String(),   // other chain sender
		rndHash,
		timestamp,
		types.Coins{{Denom: "BNB", Amount: 100_000_000}}, //{Denom: "BNB", Amount: 100_000_000}},
		"",  // expected income
		360, // heightspan
		true,
		bnbRpc.Commit,
	)
	require.NoError(t, err)
	createMsg := bep3.NewMsgCreateAtomicSwap(
		common.KavaDeputyAddr,           // sender
		common.KavaUserAddrs[0],         // recipient
		common.BnbDeputyAddr.String(),   // recipient other chain
		common.BnbUserAddrs[0].String(), // sender other chain
		rndHash,
		timestamp,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 100_000_000)),
		250,
	)
	require.NoError(t, createMsg.ValidateBasic())
	res, err := kavaClient.Broadcast(createMsg, client.Commit)
	require.NoError(t, err)
	require.EqualValues(t, res.Code, 0)

	// Create another pair of swaps
	rndNum2, err := bep3.GenerateSecureRandomNumber()
	require.NoError(t, err)
	rndHash2 := bep3.CalculateRandomHash(rndNum2, timestamp)
	_, err = bnbClient.HTLT(
		common.BnbDeputyAddr,             // recipient
		common.KavaUserAddrs[0].String(), // recipient other chain
		common.KavaDeputyAddr.String(),   // other chain sender
		rndHash2,
		timestamp,
		types.Coins{{Denom: "BNB", Amount: 100_000_000}}, //{Denom: "BNB", Amount: 100_000_000}},
		"",  // expected income
		360, // heightspan
		true,
		bnbRpc.Commit,
	)
	require.NoError(t, err)
	createMsg2 := bep3.NewMsgCreateAtomicSwap(
		common.KavaDeputyAddr,           // sender
		common.KavaUserAddrs[0],         // recipient
		common.BnbDeputyAddr.String(),   // recipient other chain
		common.BnbUserAddrs[0].String(), // sender other chain
		rndHash2,
		timestamp,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 100_000_000)),
		250,
	)
	require.NoError(t, createMsg2.ValidateBasic())
	res, err = kavaClient.Broadcast(createMsg2, client.Commit)
	require.EqualValues(t, res.Code, 0)
	require.NoError(t, err)

	// Create another pair of swaps
	rndNum3, err := bep3.GenerateSecureRandomNumber()
	require.NoError(t, err)
	rndHash3 := bep3.CalculateRandomHash(rndNum3, timestamp)
	_, err = bnbClient.HTLT(
		common.BnbDeputyAddr,             // recipient
		common.KavaUserAddrs[0].String(), // recipient other chain
		common.KavaDeputyAddr.String(),   // other chain sender
		rndHash3,
		timestamp,
		types.Coins{{Denom: "BNB", Amount: 100_000_000}}, //{Denom: "BNB", Amount: 100_000_000}},
		"",  // expected income
		360, // heightspan
		true,
		bnbRpc.Commit,
	)
	require.NoError(t, err)
	createMsg3 := bep3.NewMsgCreateAtomicSwap(
		common.KavaDeputyAddr,           // sender
		common.KavaUserAddrs[0],         // recipient
		common.BnbDeputyAddr.String(),   // recipient other chain
		common.BnbUserAddrs[0].String(), // sender other chain
		rndHash3,
		timestamp,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 100_000_000)),
		250,
	)
	require.NoError(t, createMsg3.ValidateBasic())
	res, err = kavaClient.Broadcast(createMsg3, client.Commit)
	require.EqualValues(t, res.Code, 0)
	require.NoError(t, err)

	// claim first two kava htlts
	time.Sleep(3 * time.Second)
	kavaID := bep3.CalculateSwapID(rndHash, common.KavaDeputyAddr, common.BnbUserAddrs[0].String())
	claimMsg := bep3.NewMsgClaimAtomicSwap(
		common.KavaDeputyAddr,
		kavaID,
		rndNum,
	)
	res, err = kavaClient.Broadcast(claimMsg, client.Commit)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.Code)

	kavaID2 := bep3.CalculateSwapID(rndHash2, common.KavaDeputyAddr, common.BnbUserAddrs[0].String())
	claimMsg2 := bep3.NewMsgClaimAtomicSwap(
		common.KavaDeputyAddr,
		kavaID2,
		rndNum2,
	)
	res, err = kavaClient.Broadcast(claimMsg2, client.Commit)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.Code)

	// run
	time.Sleep(5 * time.Second) // TODO replace with wait func
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewBnbClaimer("http://localhost:1317", "tcp://localhost:26657", "tcp://localhost:26658", common.KavaDeputyAddr.String(), common.BnbDeputyAddr.String(), common.BnbUserMnemonics[:2]).Run(ctx)
	defer shutdownClaimer()
	time.Sleep(20 * time.Second) // TODO replace with waiting for swaps to be claimed (with timeout)

	// check the first two bnb swap were claimed
	bnbSwapID := msg.CalculateSwapID(rndHash, common.BnbUserAddrs[0], common.KavaDeputyAddr.String())
	s, err := bnbClient.GetSwapByID(bnbSwapID)
	require.NoError(t, err)
	require.Equal(t, types.Completed, s.Status)

	bnbSwapID2 := msg.CalculateSwapID(rndHash2, common.BnbUserAddrs[0], common.KavaDeputyAddr.String())
	s2, err := bnbClient.GetSwapByID(bnbSwapID2)
	require.NoError(t, err)
	require.Equal(t, types.Completed, s2.Status)
}
