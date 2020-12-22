// +build integration

package integration

import (
	"context"
	"math"
	"testing"
	"time"

	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-tools/deputy-claimer/test/addresses"
	"github.com/kava-labs/go-tools/deputy-claimer/test/swap"
	kavatypes "github.com/kava-labs/kava/x/bep3/types"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/renamethis"
)

func TestClaimConcurrentSwapsKava(t *testing.T) {

	addrs := addresses.GetAddresses()

	kavaSwapper := swap.NewKavaSwapClient(addresses.KavaNodeURL)
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
		_, err := kavaSwapper.Create(swap.KavaSwap, client.Commit)
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
			Endpoint:  addresses.KavaNodeURL,
			Mnemonics: kavaUserMenmonics(addrs)[2:],
		},
		BinanceChain: config.BinanceChainConfig{
			ChainID:  "Binance-Chain-Tigris",
			Endpoint: addresses.BnbNodeURL,
			Mnemonic: bnbUserMenmonics(addrs)[0],
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go renamethis.Main(ctx, cfg)
	time.Sleep(1 * time.Second) // give time for the server to start

	for _, s := range swaps {
		err := sendClaimRequest("kava", s.KavaSwap.GetSwapID(), s.RandomNumber)
		require.NoError(t, err)
	}
	averageBlockTime := 6
	waitTime := int(math.Ceil(
		float64(numConcurrentSwaps)/float64(len(cfg.Kava.Mnemonics)),
	)) * averageBlockTime
	time.Sleep(time.Duration(waitTime) * time.Second)

	for _, s := range swaps {
		status, err := kavaSwapper.FetchStatus(s.KavaSwap)
		require.NoError(t, err)
		require.Equalf(t, kavatypes.Completed, status, "expected swap status '%s', actual '%s'", kavatypes.Completed, status)
	}
}
