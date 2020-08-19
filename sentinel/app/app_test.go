package app

import (
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {

	config := sdk.GetConfig()
	app.SetBech32AddressPrefixes(config)
	app.SetBip44CoinType(config)
	config.Seal()

	os.Exit(m.Run())
}
func TestCalculateDebtAdjustment(t *testing.T) {
	adjustment := calculateDebtAdjustment(d("1.8"), i(100_000_000_000), d("2.25"))

	require.Equal(t, i(-20_000_000_000), adjustment)
}
func TestTargetRatio(t *testing.T) {
	app := App{
		lowerTrigger: d("2.0"),
		upperTrigger: d("2.5"),
	}
	require.Equal(t, d("2.25"), app.targetRatio())
}
func TestTxSigner_Sign(t *testing.T) {
	// 1) setup
	testMnemonic := "very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food"
	signer := NewDefaultTxSigner(testMnemonic)

	var stdTx authtypes.StdTx
	var sequence, accountNumber uint64
	chainID := "a-chain-id"

	// 2) create signature
	sig, err := signer.Sign(stdTx, accountNumber, sequence, chainID)
	require.NoError(t, err, "asfasfasdf")

	// 3) verify results
	info, err := signer.keybase.Get(signer.keybaseKeyName)
	require.NoError(t, err)
	sigCorrect := info.GetPubKey().VerifyBytes(
		authtypes.StdSignBytes(chainID, accountNumber, sequence, stdTx.Fee, stdTx.Msgs, stdTx.Memo),
		sig.Signature,
	)
	require.True(t, sigCorrect)
}

func TestTxSigner_GetAddress(t *testing.T) {
	testMnemonic := "very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food"
	testAddress, err := sdk.AccAddressFromBech32("kava1ypjp0m04pyp73hwgtc0dgkx0e9rrydecm054da")
	require.NoError(t, err)
	signer := NewDefaultTxSigner(testMnemonic)

	address := signer.GetAddress()

	require.Equal(t, testAddress, address)
}

func d(float string) sdk.Dec                { return sdk.MustNewDecFromStr(float) }
func i(n int64) sdk.Int                     { return sdk.NewInt(n) }
func c(denom string, amount int64) sdk.Coin { return sdk.NewInt64Coin(denom, amount) }
func cs(coins ...sdk.Coin) sdk.Coins        { return sdk.NewCoins(coins...) }
