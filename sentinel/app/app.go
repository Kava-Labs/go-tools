package app

import (
	"errors"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
)

const (
	defaultWaitPeriod = 2 * time.Second
	numFetchAttempts  = 2
)

var defaultFee = authtypes.NewStdFee(
	500_000,
	sdk.NewCoins(sdk.NewCoin("ukava", sdk.ZeroInt())),
)

type App struct {
	client       Client
	signer       TxSigner
	logger       Logger
	cdpDenom     string
	lowerTrigger sdk.Dec
	upperTrigger sdk.Dec
	waitPeriod   time.Duration
	chainID      string
	txFee        authtypes.StdFee
}

func NewApp(client Client, signer TxSigner, logger Logger, cdpDenom string, lowerTrigger, upperTrigger sdk.Dec, waitPeriod time.Duration, chainID string, fee authtypes.StdFee) (App, error) {
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
		logger:       logger,
		cdpDenom:     cdpDenom,
		lowerTrigger: lowerTrigger,
		upperTrigger: upperTrigger,
		waitPeriod:   waitPeriod,
		chainID:      chainID,
		txFee:        fee,
	}, nil
}

// NewDefaultApp is a convenience function that returns an app with some configuration filled in with defaults.
func NewDefaultApp(logger Logger, restURL string, cdpOwnerMnemonic, cdpDenom, chainID string, lowerTrigger, upperTrigger sdk.Dec) (App, error) {
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
		logger,
		cdpDenom,
		lowerTrigger,
		upperTrigger,
		defaultWaitPeriod,
		chainID,
		defaultFee,
	)
}

// Run is the main entrypoint for the App
func (app App) Run() {
	if err := app.RunPreemptiveValidation(); err != nil {
		app.logger.Fatalf("could not validate app config:", err) // fatal
	}

	for {
		if err := app.AttemptRebalanceCDP(); err != nil {
			app.logger.Printf("unexpected error rebalancing cdp:", err) // send alert
		}
		time.Sleep(app.waitPeriod)
	}
}

// AttemptRebalanceCDP performs tries to move a cdp to the target collateral ratio by repaying or withdrawing debt.
func (app App) AttemptRebalanceCDP() error {
	// TODO ensure full node is not syncing
	state, err := app.fetchChainState()
	if err != nil {
		return err
	}

	if isWithinRange(state.cdp.CollateralizationRatio, app.lowerTrigger, app.upperTrigger) {
		app.logger.Printf("collateral ratio (%s) has not deviated enough from target (%s)", state.cdp.CollateralizationRatio, app.targetRatio())
		return nil
	}
	desiredDebtChange := calculateDebtAdjustment(
		state.cdp.CollateralizationRatio,
		totalPrinciple(state.cdp.CDP).Amount,
		app.targetRatio(),
	)
	if desiredDebtChange.Equal(sdk.ZeroInt()) {
		app.logger.Println("amount to rebalance is 0")
		return nil
	}

	msg := app.constructMsg(desiredDebtChange, state.cdp.Principal.Denom)

	stdTx, err := constructSignedStdTx(app.signer, msg, state.account.GetAccountNumber(), state.account.GetSequence(), app.chainID, app.txFee)
	if err != nil {
		return fmt.Errorf("could not create tx: %w", err)
	}
	err = app.client.broadcastTx(stdTx)
	var e *MempoolRejectionError
	if errors.As(err, &e) {
		app.logger.Println(e.Error())
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not broadcast tx: %w", err)
	}

	app.logger.Printf("successfully submitted tx %s for %s debt", msg.Type(), desiredDebtChange)
	return nil
}

type chainState struct {
	cdp     cdptypes.AugmentedCDP
	account authexported.Account
	height  int64
}

func (app App) fetchChainState() (chainState, error) {
	fetch := func() (chainState, error) {
		augmentedCDP, heightCDP, err := app.client.getAugmentedCDP(app.cdpOwner(), app.cdpDenom)
		if err != nil {
			return chainState{}, fmt.Errorf("could not fetch cdp: %w", err)
		}
		account, heightAcc, err := app.client.getAccount(app.cdpOwner())
		if err != nil {
			return chainState{}, fmt.Errorf("could not fetch cdp owner account: %w", err)
		}
		if heightCDP != heightAcc {
			return chainState{}, fmt.Errorf("height mismatch")
		}
		return chainState{cdp: augmentedCDP, account: account, height: heightCDP}, nil
	}

	var state chainState
	var err error
	for attempts := numFetchAttempts; attempts > 0; attempts-- {
		state, err = fetch()
		if err == nil {
			break
		}
	}
	return state, err
}

// RunPreemptiveValidation performs some checks to ensure errors will not occur at the critical point when debt needs to be repayed.
func (app App) RunPreemptiveValidation() error {

	// check chain id matches chain
	chainID, err := app.client.getChainID()
	if err != nil {
		return fmt.Errorf("could not fetch chainID: %w", err)
	}
	if chainID != app.chainID {
		return fmt.Errorf("config chain ID (%s) doesn't match node's chain ID (%s)", app.chainID, chainID)
	}

	// check account has enough stable coin
	augmentedCDP, _, err := app.client.getAugmentedCDP(app.cdpOwner(), app.cdpDenom)
	if err != nil {
		return fmt.Errorf("could not fetch cdp: %w", err)
	}
	account, _, err := app.client.getAccount(app.cdpOwner())
	if err != nil {
		return fmt.Errorf("could not fetch cdp owner account: %w", err)
	}
	coins := account.SpendableCoins(time.Now()) // TODO fetch latest block time for more accuracy
	stableCoinBalance := coins.AmountOf(augmentedCDP.Principal.Denom)
	if !stableCoinBalance.IsPositive() { // TODO trigger error based on max price movement covered by stable balance
		return fmt.Errorf("cdp owner account holds no stable coin")
	}
	// TODO check there is balance to pay fees, feeCoinBalance := coins.AmountOf(app.txFee.Fee.Denom)

	// check lower trigger is safe
	params, _, err := app.client.getCDPParams()
	if err != nil {
		return fmt.Errorf("could not fetch cdp params: %w", err)
	}
	var collateralParam cdptypes.CollateralParam
	for _, cp := range params.CollateralParams {
		if cp.Denom == app.cdpDenom {
			collateralParam = cp
		}
	}
	maxPctPriceChangePerBlock := sdk.MustNewDecFromStr("0.1") // TODO check against historical price movements, and block times
	minTriggerRatio := collateralParam.LiquidationRatio.Quo(sdk.OneDec().Sub(maxPctPriceChangePerBlock))
	if app.lowerTrigger.LT(minTriggerRatio) {
		return fmt.Errorf("lower trigger (%s) below safe limit (%s) based on historic price changes", app.lowerTrigger, minTriggerRatio)
	}

	return nil
}

func (app App) cdpOwner() sdk.AccAddress {
	return app.signer.GetAddress()
}

// targetRatio is the collateral ratio that the CDP will be adjusted to.
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

// calculateDebtAdjustment returns how much a CDPs debt should be changed to reach a target collateral ratio
func calculateDebtAdjustment(currentCollatRatio sdk.Dec, currentDebt sdk.Int, targetCollatRatio sdk.Dec) sdk.Int {
	// currentRatio * currentDebt == desiredRatio * desiredDebt
	preciseDesiredDebt := currentCollatRatio.MulInt(currentDebt).Quo(targetCollatRatio)
	// round the debt down so that it will never result in a ratio lower than target
	desiredDebt := preciseDesiredDebt.TruncateInt()

	return desiredDebt.Sub(currentDebt)
}

// isWithinRange checks if a number is between two values, excluding endpoints.
func isWithinRange(number, rangeMin, rangeMax sdk.Dec) bool {
	return number.GT(rangeMin) && number.LT(rangeMax)
}

// calculateMidPoint returns a number halfway between two numbers
func calculateMidPoint(a, b sdk.Dec) sdk.Dec {
	two := sdk.MustNewDecFromStr("2.0")
	return a.Quo(two).Add(b.Quo(two))
}
