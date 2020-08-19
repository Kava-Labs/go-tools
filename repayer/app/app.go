package app

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
)

const (
	stableCoinDenom = "usdx"
	kavaDenom       = "ukava"
	defaultGas      = 300_000
)

type App struct {
	client   Client
	signer   TxSigner
	cdpDenom string
	chainID  string
}

func NewApp(restURL string, cdpOwnerMnemonic, cdpDenom, chainID string) App {
	return App{
		client:   NewClient(restURL),
		signer:   NewDefaultTxSigner(cdpOwnerMnemonic),
		cdpDenom: cdpDenom,
		chainID:  chainID,
	}
}
func (app App) Run() error {
	augmentedCDP, heightCDP, err := app.client.getAugmentedCDP(app.cdpOwner(), app.cdpDenom)
	if err != nil {
		return err
	}
	account, heightAcc, err := app.client.getAccount(app.cdpOwner())
	if err != nil {
		return err
	}
	if heightCDP != heightAcc {
		return fmt.Errorf("mismatched queried state") // TODO above errors are immediate retry
	}

	desiredDebtChange := calculateDebtAdjustment(
		augmentedCDP.CollateralizationRatio,
		totalPrinciple(augmentedCDP.CDP).Amount,
		sdk.MustNewDecFromStr("2.25"),
	)
	// TODO only submit tx if debt change is large enough
	fmt.Println("change: ", desiredDebtChange)

	msg := app.constructMsg(desiredDebtChange)

	stdTx, err := constructSignedStdTx(app.signer, msg, account.GetAccountNumber(), account.GetSequence(), app.chainID)
	if err != nil {
		return err // TODO log, something has gone wrong, untemporarily
	}
	fmt.Println("tx: ", stdTx)
	err = app.client.broadcastTx(stdTx)
	if err != nil {
		return err // TODO expect errors about tx being rejected, log fatal errors, other wise retry after timeout
	}

	time.Sleep(8 * time.Second)
	hash, err := txHash(app.client.codec, stdTx)
	if err != nil {
		return err
	}
	res, err := app.client.getTx(hash)
	if err != nil {
		return err
	}
	fmt.Println("result: ", res)
	// TODO add loop, basic integration test, tidy up

	return nil
}

func (app App) cdpOwner() sdk.AccAddress {
	return app.signer.GetAddress()
}

func (app App) constructMsg(desiredDebtChange sdk.Int) sdk.Msg {
	if desiredDebtChange.IsNegative() {
		desiredDebtChange = desiredDebtChange.MulRaw(-1) // get positive amount
		return cdptypes.NewMsgRepayDebt(app.cdpOwner(), app.cdpDenom, sdk.NewCoin(stableCoinDenom, desiredDebtChange))
	} else {
		return cdptypes.NewMsgDrawDebt(app.cdpOwner(), app.cdpDenom, sdk.NewCoin(stableCoinDenom, desiredDebtChange))
	}
}
func constructSignedStdTx(signer TxSigner, msg sdk.Msg, accountNumber, sequence uint64, chainID string) (authtypes.StdTx, error) {
	stdTx := authtypes.StdTx{
		Msgs: []sdk.Msg{msg},
		Fee: authtypes.NewStdFee(
			defaultGas,
			sdk.NewCoins(sdk.NewCoin(kavaDenom, sdk.ZeroInt())),
		),
		Memo: "",
	}
	sig, err := signer.Sign(stdTx, accountNumber, sequence, chainID)
	if err != nil {
		return authtypes.StdTx{}, err
	}
	stdTx.Signatures = append(stdTx.Signatures, sig)

	return stdTx, nil
}

func totalPrinciple(cdp cdptypes.CDP) sdk.Coin {
	return cdp.Principal.Add(cdp.AccumulatedFees)
}
func calculateDebtAdjustment(currentCollatRatio sdk.Dec, currentDebt sdk.Int, targetCollatRatio sdk.Dec) sdk.Int {
	// currentRatio*currentDebt == desiredRatio * desiredDebt
	preciseDesiredDebt := currentCollatRatio.MulInt(currentDebt).Quo(targetCollatRatio)
	// round the debt down so that it will never result in a ratio lower than target
	desiredDebt := preciseDesiredDebt.TruncateInt()

	return desiredDebt.Sub(currentDebt)
}
