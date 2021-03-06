// +build integration

package app

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/sentinel/integration_test/common"
)

var distantFuture = time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)

func TestGetAccount(t *testing.T) {
	client, err := NewClient(common.KavaRestURL)
	require.NoError(t, err)

	account, _, err := client.getAccount(common.KavaUserAddrs[0])
	require.NoError(t, err)

	require.Equal(t, account.GetAddress(), common.KavaUserAddrs[0])
}
func TestGetAugmentedCDP(t *testing.T) {
	client, err := NewClient(common.KavaRestURL)
	require.NoError(t, err)
	owner := common.KavaUserAddrs[0]
	denom := "bnb"

	augmentedCDP, _, err := client.getAugmentedCDP(owner, denom)
	require.NoError(t, err)

	require.Equal(t, uint64(1), augmentedCDP.CDP.ID)
}
func TestGetCDPParams(t *testing.T) {
	client, err := NewClient(common.KavaRestURL)
	require.NoError(t, err)

	params, _, err := client.getCDPParams()
	require.NoError(t, err)

	require.Equal(t, "bnb", params.CollateralParams[0].Denom)
}

func TestGetChainID(t *testing.T) {
	client, err := NewClient(common.KavaRestURL)
	require.NoError(t, err)

	chainID, err := client.getChainID()
	require.NoError(t, err)

	require.Equal(t, common.KavaChainID, chainID)
}

func TestBroadcastAndGetTx(t *testing.T) {
	client, err := NewClient(common.KavaRestURL)
	require.NoError(t, err)

	msg := bank.NewMsgSend(common.KavaUserAddrs[0], common.KavaUserAddrs[1], cs(c(common.KavaGovDenom, 1)))
	account, _, err := client.getAccount(common.KavaUserAddrs[0])
	require.NoError(t, err)
	signer, err := NewDefaultTxSigner(common.KavaUserMnemonics[0])
	require.NoError(t, err)
	stdTx, err := constructSignedStdTx(signer, msg, account.GetAccountNumber(), account.GetSequence(), common.KavaChainID, defaultFee)
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

func TestApp_Run(t *testing.T) {
	cdpOwner := common.KavaUserMnemonics[0]
	cdpDenom := "bnb"
	app, err := NewDefaultApp(NewDefaultLogger(), common.KavaRestURL, cdpOwner, cdpDenom, common.KavaChainID, d("2.00"), d("2.5"))
	require.NoError(t, err)

	// cdp is at certain ratio
	augmentedCDP, _, err := app.client.getAugmentedCDP(app.cdpOwner(), cdpDenom)
	require.NoError(t, err)
	t.Log(augmentedCDP) // TODO verify cdp is not at target ratio

	// run app
	err = app.AttemptRebalanceCDP()
	require.NoError(t, err)
	time.Sleep(6 * time.Second) // wait until tx is in block

	// check cdp is at desired ratio
	augmentedCDP, _, err = app.client.getAugmentedCDP(app.cdpOwner(), cdpDenom)
	require.NoError(t, err)
	t.Log(augmentedCDP)

	// check cdp has been repayed to within a percentage of the target rate
	acceptableError := d("0.01")
	require.Truef(t,
		app.targetRatio().Sub(augmentedCDP.CollateralizationRatio).Abs().LTE(acceptableError),
		"difference between target ratio (%s) and actual ratio (%s) is > %s",
		app.targetRatio(), augmentedCDP.CollateralizationRatio, acceptableError,
	)
}

func TestApp_Concurrent(t *testing.T) {
	cdpOwner := common.KavaUserMnemonics[1]
	cdpDenom := "bnb"
	numReplicas := 100

	// setup many copies of app
	apps := make(chan App, numReplicas)
	var wg sync.WaitGroup
	for i := 0; i < numReplicas; i++ {
		wg.Add(1)
		go func(apps chan App) {
			defer wg.Done()
			app, err := NewDefaultApp(NewDefaultLogger(), common.KavaRestURL, cdpOwner, cdpDenom, common.KavaChainID, d("2.00"), d("2.5"))
			require.NoError(t, err)
			apps <- app
		}(apps)
	}
	wg.Wait()

	// run all the apps at once
	close(apps)
	for a := range apps {
		go func(a App) {
			err := a.AttemptRebalanceCDP()
			require.NoError(t, err)
		}(a)
	}
	time.Sleep(8 * time.Second) // wait until tx is in block

	// check cdp is at desired ratio
	verificationApp, err := NewDefaultApp(NewDefaultLogger(), common.KavaRestURL, cdpOwner, cdpDenom, common.KavaChainID, d("2.00"), d("2.5"))
	require.NoError(t, err)
	augmentedCDP, _, err := verificationApp.client.getAugmentedCDP(verificationApp.cdpOwner(), cdpDenom)
	require.NoError(t, err)

	// check cdp has been repayed to within a percentage of the target rate
	acceptableError := d("0.01")
	require.Truef(t,
		verificationApp.targetRatio().Sub(augmentedCDP.CollateralizationRatio).Abs().LTE(acceptableError),
		"difference between target ratio (%s) and actual ratio (%s) is > %s",
		verificationApp.targetRatio(), augmentedCDP.CollateralizationRatio, acceptableError,
	)
}
