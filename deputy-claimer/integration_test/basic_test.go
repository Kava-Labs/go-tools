// +build integration

package main

import (
	"context"
	"testing"
	"time"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/binance-chain-go-sdk/types/msg"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
	"github.com/kava-labs/go-tools/deputy-claimer/integration_test/common"
)

func TestClaimKava(t *testing.T) {
	// setup clients
	cdc := app.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, common.KavaUserMnemonics[0], app.Bip44CoinType, common.KavaNodeURL, client.LocalNetwork)
	kavaKeyM, err := kavaKeys.NewMnemonicKeyManager(common.KavaUserMnemonics[0], app.Bip44CoinType)
	require.NoError(t, err)
	bnbKeyM, err := bnbKeys.NewMnemonicKeyManager(common.BnbDeputyMnemonic)
	require.NoError(t, err)
	bnbClient := bnbRpc.NewRPCClient(common.BnbNodeURL, types.ProdNetwork)
	bnbClient.SetKeyManager(bnbKeyM)

	// send htlt on kva
	rndNum, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	timestamp := time.Now().Unix() - 10*60 - 1 // set the timestamp to be in the past
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
	createMsg := bep3types.NewMsgCreateAtomicSwap(
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
	rndNum2, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	rndHash2 := bep3types.CalculateRandomHash(rndNum2, timestamp)
	createMsg2 := bep3types.NewMsgCreateAtomicSwap(
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
	kavaSwapID := bep3types.CalculateSwapID(rndHash, kavaKeyM.GetAddr(), common.BnbDeputyAddr.String())
	s, err := kavaClient.GetSwapByID(kavaSwapID)
	require.NoError(t, err)
	require.Equal(t, bep3types.Completed, s.Status)
}

func TestClaimBnb(t *testing.T) {
	// setup clients
	cdc := app.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, common.KavaDeputyMnemonic, app.Bip44CoinType, common.KavaNodeURL, client.LocalNetwork)
	bnbKeyM, err := bnbKeys.NewMnemonicKeyManager(common.BnbUserMnemonics[0])
	require.NoError(t, err)
	bnbClient := bnbRpc.NewRPCClient(common.BnbNodeURL, types.ProdNetwork)
	bnbClient.SetKeyManager(bnbKeyM)

	// Create a swap on each chain
	rndNum, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	timestamp := time.Now().Unix() - 10*60 - 1 // set the timestamp to be in the past
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
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
	createMsg := bep3types.NewMsgCreateAtomicSwap(
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
	rndNum2, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	rndHash2 := bep3types.CalculateRandomHash(rndNum2, timestamp)
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
	createMsg2 := bep3types.NewMsgCreateAtomicSwap(
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

	// claim first kava htlt
	time.Sleep(3 * time.Second)
	kavaID := bep3types.CalculateSwapID(rndHash, common.KavaDeputyAddr, common.BnbUserAddrs[0].String())
	claimMsg := bep3types.NewMsgClaimAtomicSwap(
		common.KavaDeputyAddr,
		kavaID,
		rndNum,
	)
	res, err = kavaClient.Broadcast(claimMsg, client.Commit)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.Code)

	// run
	time.Sleep(5 * time.Second) // TODO replace with wait func
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewBnbClaimer("http://localhost:1317", "tcp://localhost:26657", "tcp://localhost:26658", common.KavaDeputyAddr.String(), common.BnbDeputyAddr.String(), common.BnbUserMnemonics[:2]).Run(ctx)
	defer shutdownClaimer()
	time.Sleep(8 * time.Second)

	// check the first bnb swap was claimed
	bnbSwapID := msg.CalculateSwapID(rndHash, common.BnbUserAddrs[0], common.KavaDeputyAddr.String())
	s, err := bnbClient.GetSwapByID(bnbSwapID)
	require.NoError(t, err)
	require.Equal(t, types.Completed, s.Status)
}
