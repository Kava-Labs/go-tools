package app

import (
	"fmt"
	"log"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
)

const defaultWaitPeriod = 2 * time.Second

var defaultFee = authtypes.NewStdFee(
	300_000,
	sdk.NewCoins(sdk.NewCoin("ukava", sdk.ZeroInt())),
)

type App struct {
	client       Client
	signer       TxSigner
	cdpDenom     string
	lowerTrigger sdk.Dec
	upperTrigger sdk.Dec
	waitPeriod   time.Duration
	chainID      string
	txFee        authtypes.StdFee
}

func NewApp(client Client, signer TxSigner, cdpDenom string, lowerTrigger, upperTrigger sdk.Dec, waitPeriod time.Duration, chainID string, fee authtypes.StdFee) (App, error) {
	if err := sdk.ValidateDenom(cdpDenom); err != nil {
		return App{}, err
	}
	if lowerTrigger.IsNil() || lowerTrigger.IsNegative() {
		return App{}, fmt.Errorf("lower trigger is invalid")
	}
	if upperTrigger.IsNil() || upperTrigger.IsNegative() {
		return App{}, fmt.Errorf("upper trigger is invalid")
	}
	if upperTrigger.LT(lowerTrigger) {
		return App{}, fmt.Errorf("upper trigger lower than lower trigger")
	}
	if waitPeriod < 0 {
		return App{}, fmt.Errorf("invalid wait period")
	}
	return App{
		client:       client,
		signer:       signer,
		cdpDenom:     cdpDenom,
		lowerTrigger: lowerTrigger,
		upperTrigger: upperTrigger,
		waitPeriod:   waitPeriod,
		chainID:      chainID,
		txFee:        fee,
	}, nil
}

func NewDefaultApp(restURL string, cdpOwnerMnemonic, cdpDenom, chainID string, lowerTrigger, upperTrigger sdk.Dec) (App, error) {
	client, err := NewClient(restURL)
	if err != nil {
		return App{}, err
	}
	signer, err := NewDefaultTxSigner(cdpOwnerMnemonic)
	if err != nil {
		return App{}, err
	}
	return NewApp(
		client,
		signer,
		cdpDenom,
		lowerTrigger,
		upperTrigger,
		defaultWaitPeriod,
		chainID,
		defaultFee,
	)
}

// Validate
func (app App) PreFlightCheck() error {
	// check cdp exists - already checked in RunOnce

	// check fee balance - how many txs can it send
	// check usdx balance - how big a price movement can this sustain
	// Check chain id
	// check lower trigger is not within dangerous of liquidation ratio - assumptions about block time
	// params, err := app.client.getCDPParams()
	return nil
}

func (app App) Run() {
	for {
		if err := app.RunOnce(); err != nil {
			log.Println(err)
		}
		time.Sleep(app.waitPeriod)
	}
}

// TODO AdjustCDP
func (app App) RunOnce() error {
	augmentedCDP, heightCDP, err := app.client.getAugmentedCDP(app.cdpOwner(), app.cdpDenom)
	if err != nil {
		return err
	}
	account, heightAcc, err := app.client.getAccount(app.cdpOwner())
	if err != nil {
		return err
	}
	if heightCDP != heightAcc {
		return fmt.Errorf("mismatched queried state")
	}

	if isWithinRange(augmentedCDP.CollateralizationRatio, app.lowerTrigger, app.upperTrigger) {
		return fmt.Errorf("ratio has not deviated enough from target, skipping")
	}
	desiredDebtChange := calculateDebtAdjustment(
		augmentedCDP.CollateralizationRatio,
		totalPrinciple(augmentedCDP.CDP).Amount,
		app.targetRatio(),
	)
	if desiredDebtChange.Equal(sdk.ZeroInt()) {
		return fmt.Errorf("amount to send is 0, skipping")
	}

	msg := app.constructMsg(desiredDebtChange, augmentedCDP.Principal.Denom)

	stdTx, err := constructSignedStdTx(app.signer, msg, account.GetAccountNumber(), account.GetSequence(), app.chainID, app.txFee)
	if err != nil {
		return err
	}
	err = app.client.broadcastTx(stdTx)
	if err != nil {
		return err
	}
	return nil
}

func (app App) cdpOwner() sdk.AccAddress {
	return app.signer.GetAddress()
}

func (app App) targetRatio() sdk.Dec {
	return calculateMidPoint(app.lowerTrigger, app.upperTrigger)
}

func (app App) constructMsg(desiredDebtChange sdk.Int, stableCoinDenom string) sdk.Msg {
	if desiredDebtChange.IsNegative() {
		desiredDebtChange = desiredDebtChange.MulRaw(-1) // get positive amount
		return cdptypes.NewMsgRepayDebt(app.cdpOwner(), app.cdpDenom, sdk.NewCoin(stableCoinDenom, desiredDebtChange))
	} else {
		return cdptypes.NewMsgDrawDebt(app.cdpOwner(), app.cdpDenom, sdk.NewCoin(stableCoinDenom, desiredDebtChange))
	}
}

func constructSignedStdTx(signer TxSigner, msg sdk.Msg, accountNumber, sequence uint64, chainID string, fee authtypes.StdFee) (authtypes.StdTx, error) {
	stdTx := authtypes.StdTx{
		Msgs: []sdk.Msg{msg},
		Fee:  fee,
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
	// currentRatio * currentDebt == desiredRatio * desiredDebt
	preciseDesiredDebt := currentCollatRatio.MulInt(currentDebt).Quo(targetCollatRatio)
	// round the debt down so that it will never result in a ratio lower than target
	desiredDebt := preciseDesiredDebt.TruncateInt()

	return desiredDebt.Sub(currentDebt)
}

func isWithinRange(number, rangeMin, rangeMax sdk.Dec) bool {
	return number.GT(rangeMin) && number.LT(rangeMax)
}

func calculateMidPoint(a, b sdk.Dec) sdk.Dec {
	two := sdk.MustNewDecFromStr("2.0")
	return a.Quo(two).Add(b.Quo(two))
}
