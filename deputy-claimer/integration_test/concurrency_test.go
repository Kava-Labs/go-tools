// +build integration

package integrationtest

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
	bep3types "github.com/kava-labs/kava/x/bep3/types"

	"github.com/kava-labs/go-sdk/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
	"github.com/kava-labs/go-tools/deputy-claimer/testcommon"
)

func TestMultipleClaimBnb(t *testing.T) {
	addrs := testcommon.GetAddresses()

	bnbSwapper := NewBnbSwapClient(testcommon.BnbNodeURL)
	kavaSwapper := NewKavaSwapClient(testcommon.KavaNodeURL)
	swapBuilder := NewDefaultSwapBuilder(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,
		addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic,
	)
	swapBuilder = swapBuilder.WithTimestamp(time.Now().Unix() - 10*60 - 1) // set the timestamp to be in the past

	createTestBnbToKavaSwap := func() (CrossChainSwap, error) {
		swap := swapBuilder.NewBnbToKavaSwap(
			addrs.Bnb.Users[0].Mnemonic,
			addrs.Kava.Users[0].Address,
			types.Coins{{Denom: "BNB", Amount: 50_000_000}},
		)
		_, err := bnbSwapper.Create(swap.BnbSwap, bnbRpc.Commit)
		if err != nil {
			return swap, err
		}
		_, err = kavaSwapper.Create(swap.KavaSwap, client.Commit)
		if err != nil {
			return swap, err
		}
		_, err = kavaSwapper.Claim(swap.KavaSwap, swap.RandomNumber, client.Commit)
		if err != nil {
			return swap, err
		}
		return swap, nil
	}

	swaps := []CrossChainSwap{}
	for i := 0; i < 6; i++ {
		t.Logf("creating test swap %d", i)
		swap, err := createTestBnbToKavaSwap()
		require.NoError(t, err)
		swaps = append(swaps, swap)
	}

	// run
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewBnbClaimer(
		testcommon.KavaRestURL,
		testcommon.KavaNodeURL,
		testcommon.BnbNodeURL,
		getDeputyAddresses(addrs),
		addrs.BnbUserMnemonics()[:2],
	).Run(ctx)
	defer shutdownClaimer()

	time.Sleep(30 * time.Second) // TODO

	// check the all the bnb swaps were claimed
	for i, s := range swaps {
		t.Logf("checking status of swap %d", i)
		status, err := bnbSwapper.FetchStatus(s.BnbSwap)
		assert.NoError(t, err)
		t.Logf("status of swap %d: %s", i, status)
		assert.Equalf(t, types.Completed, status, "expected swap status '%s', actual '%s'", types.Completed, status)
	}

}

func TestMultipleClaimKava(t *testing.T) {
	addrs := testcommon.GetAddresses()

	bnbSwapper := NewBnbSwapClient(testcommon.BnbNodeURL)
	kavaSwapper := NewKavaSwapClient(testcommon.KavaNodeURL)
	swapBuilder := NewDefaultSwapBuilder(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,
		addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic,
	)
	swapBuilder = swapBuilder.WithTimestamp(time.Now().Unix() - 10*60 - 1) // set the timestamp to be in the past

	createTestKavaToBnbSwap := func() (CrossChainSwap, error) {
		swap := swapBuilder.NewKavaToBnbSwap(
			addrs.Kava.Users[0].Mnemonic,
			addrs.Bnb.Users[0].Address,
			sdk.NewCoins(sdk.NewInt64Coin("bnb", 50_000_000)),
		)
		_, err := kavaSwapper.Create(swap.KavaSwap, client.Commit)
		if err != nil {
			return swap, err
		}
		_, err = bnbSwapper.Create(swap.BnbSwap, bnbRpc.Commit)
		if err != nil {
			return swap, err
		}
		_, err = bnbSwapper.Claim(swap.BnbSwap, swap.RandomNumber, bnbRpc.Commit)
		if err != nil {
			return swap, err
		}
		return swap, nil
	}

	swaps := []CrossChainSwap{}
	for i := 0; i < 6; i++ {
		t.Logf("creating test swap %d", i)
		swap, err := createTestKavaToBnbSwap()
		require.NoError(t, err)
		swaps = append(swaps, swap)
	}

	// run
	ctx, shutdownClaimer := context.WithCancel(context.Background())
	claim.NewKavaClaimer(
		testcommon.KavaRestURL,
		testcommon.KavaNodeURL,
		testcommon.BnbNodeURL,
		getDeputyAddresses(addrs),
		addrs.KavaUserMnemonics()[:2],
	).Run(ctx)
	defer shutdownClaimer()

	time.Sleep(30 * time.Second) // TODO

	// check the all the bnb swaps were claimed
	for i, s := range swaps {
		t.Logf("checking status of swap %d", i)
		status, err := kavaSwapper.FetchStatus(s.KavaSwap)
		assert.NoError(t, err)
		t.Logf("status of swap %d: %s", i, status)
		assert.Equalf(t, bep3types.Completed, status, "expected swap status '%s', actual '%s'", bep3types.Completed, status)
	}

}
