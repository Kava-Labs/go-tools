//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
	"github.com/kava-labs/go-tools/deputy-claimer/test/addresses"
	"github.com/kava-labs/go-tools/deputy-claimer/test/swap"
)

func TestMain(m *testing.M) {
	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	config.Seal()

	os.Exit(m.Run())
}

func TestClaimBnb(t *testing.T) {
	addrs := addresses.GetAddresses()

	bnbSwapper := swap.NewBnbSwapClient(addresses.BnbNodeURL)
	kavaSwapper := swap.NewKavaSwapClient(addresses.KavaGrpcURL)
	swapBuilder := swap.NewDefaultSwapBuilder(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,
		addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic,
	)
	swapBuilder = swapBuilder.WithTimestamp(time.Now().Unix() - 10*60 - 1) // set the timestamp to be in the past

	// create a swap
	swap1 := swapBuilder.NewBnbToKavaSwap(
		addrs.Bnb.Users[0].Mnemonic,
		addrs.Kava.Users[0].Address,
		types.Coins{{Denom: "BNB", Amount: 500_000_000}},
	)
	_, err := bnbSwapper.Create(swap1.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)
	_, err = kavaSwapper.Create(swap1.KavaSwap, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)

	// create another swap
	swap2 := swapBuilder.NewBnbToKavaSwap(
		addrs.Bnb.Users[0].Mnemonic,
		addrs.Kava.Users[0].Address,
		types.Coins{{Denom: "BNB", Amount: 500_000_000}},
	)
	_, err = bnbSwapper.Create(swap2.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)
	_, err = kavaSwapper.Create(swap2.KavaSwap, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)

	// claim kava side of first swap
	_, err = kavaSwapper.Claim(swap1.KavaSwap, swap1.RandomNumber, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)

	// run
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewBnbClaimer(
		addresses.KavaGrpcURL,
		addresses.BnbNodeURL,
		getDeputyAddresses(addrs),
		addrs.BnbUserMnemonics()[:2],
	).Start(ctx)
	defer shutdownClaimer()
	time.Sleep(8 * time.Second)

	// check the first bnb swap was claimed
	status, err := bnbSwapper.FetchStatus(swap1.BnbSwap)
	require.NoError(t, err)
	require.Equalf(t, types.Completed, status, "expected swap status '%s', actual '%s'", types.Completed, status)
}

func TestClaimKava(t *testing.T) {
	addrs := addresses.GetAddresses()

	bnbSwapper := swap.NewBnbSwapClient(addresses.BnbNodeURL)
	kavaSwapper := swap.NewKavaSwapClient(addresses.KavaNodeURL)
	swapBuilder := swap.NewDefaultSwapBuilder(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,
		addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic,
	)
	swapBuilder = swapBuilder.WithTimestamp(time.Now().Unix() - 10*60 - 1) // set the timestamp to be in the past

	// create a swap
	swap1 := swapBuilder.NewKavaToBnbSwap(
		addrs.Kava.Users[0].Mnemonic,
		addrs.Bnb.Users[0].Address,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 50_000_000)),
	)
	_, err := kavaSwapper.Create(swap1.KavaSwap, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)
	_, err = bnbSwapper.Create(swap1.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)

	// create another swap
	swap2 := swapBuilder.NewKavaToBnbSwap(
		addrs.Kava.Users[0].Mnemonic,
		addrs.Bnb.Users[0].Address,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 50_000_000)),
	)
	_, err = kavaSwapper.Create(swap2.KavaSwap, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)
	_, err = bnbSwapper.Create(swap2.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)

	// claim bnb side of first swap
	_, err = bnbSwapper.Claim(swap1.BnbSwap, swap1.RandomNumber, bnbRpc.Commit)
	require.NoError(t, err)

	// run
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewKavaClaimer(
		addresses.KavaGrpcURL,
		addresses.BnbNodeURL,
		getDeputyAddresses(addrs),
		addrs.KavaUserMnemonics()[:2],
	).Start(ctx)
	defer shutdownClaimer()
	time.Sleep(8 * time.Second)

	// check the first kava swap was claimed
	status, err := kavaSwapper.FetchStatus(swap1.KavaSwap)
	require.NoError(t, err)
	require.Equalf(
		t,
		bep3types.SWAP_STATUS_COMPLETED, status,
		"expected swap status '%s', actual '%s'",
		bep3types.SWAP_STATUS_COMPLETED, status,
	)
}

func getDeputyAddresses(addrs addresses.Addresses) claim.DeputyAddresses {
	return claim.DeputyAddresses{
		"bnb": {
			Kava: addrs.Kava.Deputys.Bnb.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Bnb.HotWallet.Address,
		},
		"busd": {
			Kava: addrs.Kava.Deputys.Busd.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Busd.HotWallet.Address,
		},
		"btcb": {
			Kava: addrs.Kava.Deputys.Btcb.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Btcb.HotWallet.Address,
		},
		"xrpb": {
			Kava: addrs.Kava.Deputys.Xrpb.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Xrpb.HotWallet.Address,
		},
	}
}
