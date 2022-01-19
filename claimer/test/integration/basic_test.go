//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/stretchr/testify/require"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/kava/app"
	kavatypes "github.com/kava-labs/kava/x/bep3/types"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
	"github.com/kava-labs/go-tools/claimer/test/addresses"
	"github.com/kava-labs/go-tools/claimer/test/swap"
)

func TestMain(m *testing.M) {
	sdkConfig := sdk.GetConfig()
	app.SetBech32AddressPrefixes(sdkConfig)

	os.Exit(m.Run())
}
func TestClaimSwapKava(t *testing.T) {

	addrs := addresses.GetAddresses()

	kavaSwapper := swap.NewKavaSwapClient(addresses.KavaGrpcURL)
	swapBuilder := swap.NewDefaultSwapBuilder(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,
		addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic,
	)

	swap1 := swapBuilder.NewBnbToKavaSwap(
		addrs.Bnb.Users[0].Mnemonic,
		addrs.Kava.Users[0].Address,
		bnbtypes.Coins{{Denom: "BNB", Amount: 500_000_000}},
	)
	// only need the receiving chain swap created
	_, err := kavaSwapper.Create(swap1.KavaSwap, txtypes.BroadcastMode_BROADCAST_MODE_BLOCK)
	require.NoError(t, err)

	cfg := config.Config{
		Kava: config.KavaConfig{
			ChainID:   "kava-localnet",
			Endpoint:  addresses.KavaGrpcURL,
			Mnemonics: kavaUserMenmonics(addrs)[2:],
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

	err = sendClaimRequest("kava", swap1.KavaSwap.GetSwapID(), swap1.RandomNumber)
	require.NoError(t, err)

	averageBlockTime := 6 * time.Second
	time.Sleep(averageBlockTime)

	status, err := kavaSwapper.FetchStatus(swap1.KavaSwap)
	require.NoError(t, err)
	require.Equalf(
		t,
		kavatypes.SWAP_STATUS_COMPLETED,
		status,
		"expected swap status '%s', actual '%s'",
		kavatypes.SWAP_STATUS_COMPLETED, status,
	)
}
func TestClaimSwapBnb(t *testing.T) {
	time.Sleep(1 * time.Second)
	addrs := addresses.GetAddresses()

	bnbSwapper := swap.NewBnbSwapClient(addresses.BnbNodeURL)
	swapBuilder := swap.NewDefaultSwapBuilder(
		addrs.Kava.Deputys.Bnb.HotWallet.Mnemonic,
		addrs.Bnb.Deputys.Bnb.HotWallet.Mnemonic,
	)

	swap1 := swapBuilder.NewKavaToBnbSwap(
		addrs.Kava.Users[0].Mnemonic,
		addrs.Bnb.Users[0].Address,
		sdk.NewCoins(sdk.NewInt64Coin("bnb", 500_000_000)),
	)
	// only need the receiving chain swap created
	_, err := bnbSwapper.Create(swap1.BnbSwap, bnbRpc.Commit)
	require.NoError(t, err)

	cfg := config.Config{
		Kava: config.KavaConfig{
			ChainID:   "kava-localnet",
			Endpoint:  addresses.KavaGrpcURL,
			Mnemonics: kavaUserMenmonics(addrs)[2:],
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

	err = sendClaimRequest("binance", swap1.BnbSwap.GetSwapID(), swap1.RandomNumber)
	require.NoError(t, err)

	averageBlockTime := 6 * time.Second
	time.Sleep(averageBlockTime)

	status, err := bnbSwapper.FetchStatus(swap1.BnbSwap)
	require.NoError(t, err)
	require.Equalf(t, bnbtypes.Completed, status, "expected swap status '%s', actual '%s'", bnbtypes.Completed, status)
}

func startApp(cfg config.Config) func() {
	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := claimer.NewDispatcher(cfg)
	go dispatcher.Start(ctx)

	s := server.NewServer(dispatcher.JobQueue())
	go s.Start()

	shutdown := func() {
		s.Shutdown(context.Background())
		cancel()
	}
	return shutdown
}

func sendClaimRequest(chain string, swapID []byte, randomNumber []byte) error {
	resp, err := http.Post(
		fmt.Sprintf(
			"http://localhost:8080/claim?target-chain=%s&swap-id=%x&random-number=%x",
			chain,
			swapID,
			randomNumber,
		),
		"",
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("claim request failed, status: %s, body: %s", resp.Status, body)
	}
	return nil
}

func kavaUserMenmonics(addrs addresses.Addresses) []string {
	mnemonics := []string{}
	for _, u := range addrs.Kava.Users {
		mnemonics = append(mnemonics, u.Mnemonic)
	}
	return mnemonics
}
func bnbUserMenmonics(addrs addresses.Addresses) []string {
	mnemonics := []string{}
	for _, u := range addrs.Bnb.Users {
		mnemonics = append(mnemonics, u.Mnemonic)
	}
	return mnemonics
}
