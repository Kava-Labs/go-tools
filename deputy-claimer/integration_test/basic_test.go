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

	bnbSwapper := NewBnbSwapClient(common.BnbNodeURL)
	kavaSwapper := NewKavaSwapClient(common.KavaNodeURL)
	swapBuilder := NewDefaultSwapBuilder(
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
	_, err = kavaSwapper.Create(swap1.KavaSwap, client.Commit)
	require.NoError(t, err)

	// create another swap
	swap2 := swapBuilder.NewBnbToKavaSwap(
		addrs.Bnb.Users[0].Mnemonic,
		addrs.Kava.Users[0].Address,
		types.Coins{{Denom: "BNB", Amount: 500_000_000}},
	)
	_, err = bnbSwapper.Create(swap2.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)
	_, err = kavaSwapper.Create(swap2.KavaSwap, client.Commit)
	require.NoError(t, err)

	// claim kava side of first swap
	_, err = kavaSwapper.Claim(swap1.KavaSwap, swap1.RandomNumber, client.Commit)
	require.NoError(t, err)

	// run
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
	status, err := bnbSwapper.FetchStatus(swap1.BnbSwap)
	require.NoError(t, err)
	require.Equalf(t, types.Completed, status, "expected swap status '%s', actual '%s'", types.Completed, status)
}

func TestClaimKava(t *testing.T) {
	addrs := common.GetAddresses()

	bnbSwapper := NewBnbSwapClient(common.BnbNodeURL)
	kavaSwapper := NewKavaSwapClient(common.KavaNodeURL)
	swapBuilder := NewDefaultSwapBuilder(
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
	_, err := kavaSwapper.Create(swap1.KavaSwap, client.Commit)
	require.NoError(t, err)
	_, err = bnbSwapper.Create(swap1.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)

	// create another swap
	swap2 := swapBuilder.NewKavaToBnbSwap(
		addrs.Kava.Users[0].Mnemonic,
		addrs.Bnb.Users[0].Address,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 50_000_000)),
	)
	_, err = kavaSwapper.Create(swap2.KavaSwap, client.Commit)
	require.NoError(t, err)
	_, err = bnbSwapper.Create(swap2.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)

	// claim bnb side of first swap
	_, err = bnbSwapper.Claim(swap1.BnbSwap, swap1.RandomNumber, bnbRpc.Commit)
	require.NoError(t, err)

	// run
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
	status, err := kavaSwapper.FetchStatus(swap1.KavaSwap)
	require.NoError(t, err)
	require.Equalf(t, bep3types.Completed, status, "expected swap status '%s', actual '%s'", bep3types.Completed, status)
}
