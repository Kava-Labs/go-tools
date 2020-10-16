// +build integration

package integrationtest

import (
	"context"
	"os"
	"testing"
	"time"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/binance-chain-go-sdk/types/msg"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
	"github.com/kava-labs/go-tools/deputy-claimer/integration_test/common"
)

func TestMain(m *testing.M) {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	config.Seal()

	os.Exit(m.Run())
}

func TestClaimBnb(t *testing.T) {
	addrs := common.GetAddresses()
	bnbClient := NewBnbClient(addrs.Bnb.Users[0].Mnemonic, common.BnbNodeURL)

	// Create a swap on each chain
	rndNum, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	timestamp := time.Now().Unix() - 10*60 - 1 // set the timestamp to be in the past
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
	bnbSwap1 := NewBnbSwap(
		addrs.Bnb.Users[0].Mnemonic,                       // sender
		addrs.Bnb.Deputys.Bnb.HotWallet.Address,           // recipient
		addrs.Kava.Deputys.Bnb.HotWallet.Address.String(), // sender other chain
		addrs.Kava.Users[0].Address.String(),              // recipient other chain
		types.Coins{{Denom: "BNB", Amount: 500_000_000}},
		timestamp,
		rndHash,
		360, // heightspan
	)
	_, err = bnbSwap1.Broadcast(bnbRpc.Commit)
	kavaSwap1 := NewKavaSwap(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,        // sender
		addrs.Kava.Users[0].Address,                      // recipient
		addrs.Bnb.Users[0].Address.String(),              // sender other chain
		addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(), // recipient other chain
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 500_000_000)),
		timestamp,
		rndHash,
		250, // heightspan
	)
	_, err = kavaSwap1.Broadcast(client.Commit)
	require.NoError(t, err)

	// Create another pair of swaps
	rndNum2, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	rndHash2 := bep3types.CalculateRandomHash(rndNum2, timestamp)
	bnbSwap2 := NewBnbSwap(
		addrs.Bnb.Users[0].Mnemonic,                       // sender
		addrs.Bnb.Deputys.Bnb.HotWallet.Address,           // recipient
		addrs.Kava.Deputys.Bnb.HotWallet.Address.String(), // sender other chain
		addrs.Kava.Users[0].Address.String(),              // recipient other chain
		types.Coins{{Denom: "BNB", Amount: 500_000_000}},
		timestamp,
		rndHash2,
		360, // heightspan
	)
	_, err = bnbSwap2.Broadcast(bnbRpc.Commit)
	require.NoError(t, err)
	kavaSwap2 := NewKavaSwap(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,        // sender
		addrs.Kava.Users[0].Address,                      // recipient
		addrs.Bnb.Users[0].Address.String(),              // sender other chain
		addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(), // recipient other chain
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 500_000_000)),
		timestamp,
		rndHash2,
		250, // heightspan
	)
	_, err = kavaSwap2.Broadcast(client.Commit)
	require.NoError(t, err)

	// claim first kava htlt
	_, err = kavaSwap1.SubmitClaim(rndNum, client.Commit)
	require.NoError(t, err)

	// run
	time.Sleep(5 * time.Second) // TODO replace with wait func
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewBnbClaimer(
		common.KavaRestURL,
		common.KavaNodeURL,
		common.BnbNodeURL,
		addrs.Kava.Deputys.Bnb.HotWallet.Address.String(),
		addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(),
		addrs.BnbUserMnemonics()[:2],
	).Run(ctx)
	defer shutdownClaimer()
	time.Sleep(8 * time.Second)

	// check the first bnb swap was claimed
	s, err := bnbClient.GetSwapByID(bnbSwap1.GetSwapID())
	require.NoError(t, err)
	require.Equal(t, types.Completed, s.Status)
}

func TestClaimKava(t *testing.T) {
	addrs := common.GetAddresses()

	// setup clients
	cdc := app.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, addrs.Kava.Users[0].Mnemonic, app.Bip44CoinType, common.KavaNodeURL)
	bnbClient := NewBnbClient(addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic, common.BnbNodeURL)

	// send htlt on kva
	rndNum, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	timestamp := time.Now().Unix() - 10*60 - 1 // set the timestamp to be in the past
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
	createMsg := bep3types.NewMsgCreateAtomicSwap(
		addrs.Kava.Users[0].Address,                      // sender
		addrs.Kava.Deputys.Bnb.HotWallet.Address,         // recipient
		addrs.Bnb.Users[0].Address.String(),              // recipient other chain
		addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(), // sender other chain
		rndHash,
		timestamp,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 100_000_000)),
		250,
	)

	_, err = kavaClient.Broadcast(createMsg, client.Commit)
	require.NoError(t, err)

	// send another htlt on kava
	rndNum2, err := bep3types.GenerateSecureRandomNumber()
	require.NoError(t, err)
	rndHash2 := bep3types.CalculateRandomHash(rndNum2, timestamp)
	createMsg2 := bep3types.NewMsgCreateAtomicSwap(
		addrs.Kava.Users[0].Address,                      // sender
		addrs.Kava.Deputys.Bnb.HotWallet.Address,         // recipient
		addrs.Bnb.Users[0].Address.String(),              // recipient other chain
		addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(), // sender other chain
		rndHash2,
		timestamp,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 100_000_000)),
		250,
	)

	_, err = kavaClient.Broadcast(createMsg2, client.Commit)
	require.NoError(t, err)

	// send corresponding htlts on bnb
	_, err = bnbClient.HTLT(
		addrs.Bnb.Users[0].Address,                        // recipient
		addrs.Kava.Deputys.Bnb.HotWallet.Address.String(), // recipient other chain
		addrs.Kava.Users[0].Address.String(),              // other chain sender
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
		addrs.Bnb.Users[0].Address,                        // recipient
		addrs.Kava.Deputys.Bnb.HotWallet.Address.String(), // recipient other chain
		addrs.Kava.Users[0].Address.String(),              // other chain sender
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
	bnbID := msg.CalculateSwapID(rndHash, addrs.Bnb.Deputys.Bnb.HotWallet.Address, addrs.Kava.Users[0].Address.String())
	_, err = bnbClient.ClaimHTLT(bnbID, rndNum, bnbRpc.Sync)
	require.NoError(t, err)

	// run
	time.Sleep(5 * time.Second) // TODO replace with wait func
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewKavaClaimer(
		common.KavaRestURL,
		common.KavaNodeURL,
		common.BnbNodeURL,
		addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(),
		addrs.KavaUserMnemonics()[:2],
	).Run(ctx)
	defer shutdownClaimer()
	time.Sleep(8 * time.Second)

	// check the first kava swap was claimed
	kavaSwapID := bep3types.CalculateSwapID(rndHash, addrs.Kava.Users[0].Address, addrs.Bnb.Deputys.Bnb.HotWallet.Address.String())
	s, err := kavaClient.GetSwapByID(kavaSwapID)
	require.NoError(t, err)
	require.Equal(t, bep3types.Completed, s.Status)
}

func NewBnbClient(mnemonic string, nodeURL string) *bnbRpc.HTTP {
	bnbKeyM, err := bnbKeys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(err)
	}
	bnbClient := bnbRpc.NewRPCClient(nodeURL, types.ProdNetwork)
	bnbClient.SetKeyManager(bnbKeyM)
	return bnbClient
}
