// +build integration

package app

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/repayer/integration_test/common"
)

func TestMain(m *testing.M) {

	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	app.SetBip44CoinType(config)
	config.Seal()

	os.Exit(m.Run())
}

func TestGetAccount(t *testing.T) {
	client := NewClient(common.KavaRestURL)

	account, err := client.getAccount(common.KavaUserAddrs[0])
	require.NoError(t, err)

	require.Equal(t, account.GetAddress(), common.KavaUserAddrs[0])
}
func TestGetCDP(t *testing.T) {
	client := NewClient(common.KavaRestURL)
	owner := common.KavaUserAddrs[0]
	denom := "bnb"

	cdp, err := client.getCDP(owner, denom)
	require.NoError(t, err)

	require.Equal(t, uint64(1), cdp.ID)
}
