//go:build integration
// +build integration

package integration

import (
	"math"
	"testing"
	"time"

	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/require"

	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	kavatypes "github.com/kava-labs/kava/x/bep3/types"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/test/addresses"
	"github.com/kava-labs/go-tools/claimer/test/swap"
)

func TestClaimConcurrentSwapsKava(t *testing.T) {

	addrs := addresses.GetAddresses()

	kavaSwapper := swap.NewKavaSwapClient(addresses.KavaGrpcURL)
	swapBuilder := swap.NewDefaultSwapBuilder(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,
		addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic,
	)

	createKavaSwap := func() (swap.CrossChainSwap, error) {
		swap := swapBuilder.NewBnbToKavaSwap(
			addrs.Bnb.Users[0].Mnemonic,
			addrs.Kava.Users[0].Address,
			bnbtypes.Coins{{Denom: "BNB", Amount: 500_000_000}},
		)
		// only need the receiving chain swap created
		_, err := kavaSwapper.Create(swap.KavaSwap, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
		if err != nil {
			return swap, err
		}
		return swap, nil
	}

	numConcurrentSwaps := 8
	swaps := []swap.CrossChainSwap{}
	for i := 0; i < numConcurrentSwaps; i++ {
		s, err := createKavaSwap()
		t.Logf("created kava swap %x", s.KavaSwap.GetSwapID())
		require.NoError(t, err)
		swaps = append(swaps, s)
	}

	cfg := config.Config{
		Kava: config.KavaConfig{
			ChainID:   "kava-localnet",
			Endpoint:  addresses.KavaGrpcURL,
			Mnemonics: kavaUserMenmonics(addrs)[1:],
		},
		BinanceChain: config.BinanceChainConfig{
			ChainID:  "Binance-Chain-Tigris",
			Endpoint: addresses.BnbNodeURL,
			Mnemonic: bnbUserMenmonics(addrs)[0],
		},
	}
	shutdownFunc := startApp(cfg)
	defer shutdownFunc()
	time.Sleep(1 * time.Second) // give time for the server to start

	for _, s := range swaps {
		err := sendClaimRequest("kava", s.KavaSwap.GetSwapID(), s.RandomNumber)
		require.NoError(t, err)
	}
	averageBlockTime := 5
	waitTime := time.Duration(
		int(math.Ceil(
			float64(numConcurrentSwaps)/float64(len(cfg.Kava.Mnemonics)),
		))*averageBlockTime+averageBlockTime,
	) * time.Second
	t.Logf("waiting for %s to let claimer claim all swaps", waitTime)
	time.Sleep(waitTime)

	for _, s := range swaps {
		status, err := kavaSwapper.FetchStatus(s.KavaSwap)
		require.NoError(t, err)
		require.Equalf(
			t,
			kavatypes.SWAP_STATUS_COMPLETED, status,
			"expected swap status '%s', actual '%s'",
			kavatypes.SWAP_STATUS_COMPLETED, status,
		)
	}
}
