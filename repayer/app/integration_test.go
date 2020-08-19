// +build integration

package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/repayer/integration_test/common"
)

var distantFuture = time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)

func TestGetAccount(t *testing.T) {
	client := NewClient(common.KavaRestURL)

	account, _, err := client.getAccount(common.KavaUserAddrs[0])
	require.NoError(t, err)

	require.Equal(t, account.GetAddress(), common.KavaUserAddrs[0])
}
func TestGetAugmentedCDP(t *testing.T) {
	client := NewClient(common.KavaRestURL)
	owner := common.KavaUserAddrs[0]
	denom := "bnb"

	augmentedCDP, _, err := client.getAugmentedCDP(owner, denom)
	require.NoError(t, err)

	require.Equal(t, uint64(1), augmentedCDP.CDP.ID)
}

func TestGetChainID(t *testing.T) {
	client := NewClient(common.KavaRestURL)

	chainID, err := client.getChainID()
	require.NoError(t, err)

	require.Equal(t, common.KavaChainID, chainID)
}

func TestGetTx(t *testing.T) {
	t.Skip()
	client := NewClient(common.KavaRestURL)

	msg := bank.NewMsgSend(common.KavaUserAddrs[0], common.KavaUserAddrs[1], cs(c(kavaDenom, 1)))
	account, _, err := client.getAccount(common.KavaUserAddrs[0])
	require.NoError(t, err)
	signer := NewDefaultTxSigner(common.KavaUserMnemonics[0])
	stdTx, err := constructSignedStdTx(signer, msg, account.GetAccountNumber(), account.GetSequence(), common.KavaChainID)
	require.NoError(t, err)

	err = client.broadcastTx(stdTx)
	require.NoError(t, err)
	time.Sleep(6 * time.Second)

	hash, err := txHash(client.codec, stdTx)
	require.NoError(t, err)

	txResponse, err := client.getTx(hash)
	require.NoError(t, err)
	require.Equal(t, strings.ToUpper(fmt.Sprintf("%x", hash)), txResponse.TxHash)
}

func TestBroadcastTx(t *testing.T) {
	t.Skip()
	client := NewClient(common.KavaRestURL)
	stdTx := authtypes.StdTx{}

	err := client.broadcastTx(stdTx)
	require.Error(t, err)

	// TODO query tx hash to check it was included on chain
}

func TestApp_Run(t *testing.T) {
	cdpOwner := common.KavaUserMnemonics[0]
	cdpDenom := "bnb"
	app := NewApp(common.KavaRestURL, cdpOwner, cdpDenom, common.KavaChainID)

	// cdp is at certain ratio
	augmentedCDP, _, err := app.client.getAugmentedCDP(app.cdpOwner(), cdpDenom)
	require.NoError(t, err)
	t.Log(augmentedCDP)

	// submit tx to change price, to change ratio
	// msg := pricefeedtypes.NewMsgPostPrice(common.KavaOracleAddr, "bnb:usd:30", d("10"), distantFuture) // TODO price: "",
	// account, _, err := app.client.getAccount(common.KavaOracleAddr)
	// require.NoError(t, err)
	// signer := NewDefaultTxSigner(common.KavaOracleMnemonic)
	// stdTx, err := constructSignedStdTx(signer, msg, account.GetAccountNumber(), account.GetSequence(), common.KavaChainID)
	// require.NoError(t, err)
	// err = app.client.broadcastTx(stdTx)
	// require.NoError(t, err)
	// time.Sleep(5 * time.Second) // TODO wait for tx to be included rather than sleeping

	// run app
	err = app.Run()
	require.NoError(t, err)

	// check cdp is at desired ratio
	augmentedCDP, _, err = app.client.getAugmentedCDP(app.cdpOwner(), cdpDenom)
	require.NoError(t, err)
	t.Log(augmentedCDP)
	targetRatio := sdk.MustNewDecFromStr("2.25") // TODO
	// check cdp has been repayed to within a percentage of the target rate
	acceptableError := d("0.01")
	require.Truef(t,
		targetRatio.Sub(augmentedCDP.CollateralizationRatio).Abs().LTE(acceptableError),
		"difference between target ratio (%s) and actual ratio (%s) is > %s",
		targetRatio, augmentedCDP.CollateralizationRatio, acceptableError,
	)
}
