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
	500_000,
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

	// currentPrice, err := app.client.getPrice("bnb:usd:30")
	// if err != nil {
	// 	return err
	// }
	// liqPrice := calculateLiquidationPrice(cdp.Collateral.Amount, cdp.GetTotalDebt().Amount, collateralParams.LiquidationRatio)
	// supportedPctPriceChange := sdk.OneDec.Sub(liqPrice.Quo(currentPrice))
	// warningPriceChange := sdk.MustNewDecFromStr("0.25")
	// if supportedPctPriceChange.LT(warningPriceChange) {
	// 	return fmt.Errorf("balance only holds enough stable coin to cover a %s%% price change", supportedPctPriceChange)
	// }

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

// func calculateLiquidationPrice(collateral, totalDebt sdk.Int, liquidationRatio sdk.Dec) sdk.Dec {
// 	return totalDebt.Quo(collateral).Mul(liquidationRatio)
// }

func (app App) Run() {
	if err := app.RunPreemptiveValidation(); err != nil {
		log.Fatal("could not validate app config:", err)
	}

	for {
		if err := app.RebalanceCDP(); err != nil {
			log.Println("did not rebalance cdp:", err)
		}
		time.Sleep(app.waitPeriod)
	}
}

func (app App) RebalanceCDP() error {
	augmentedCDP, heightCDP, err := app.client.getAugmentedCDP(app.cdpOwner(), app.cdpDenom)
	if err != nil {
		return fmt.Errorf("could not fetch cdp: %w", err)
	}
	account, heightAcc, err := app.client.getAccount(app.cdpOwner())
	if err != nil {
		return fmt.Errorf("could not fetch cdp owner account: %w", err)
	}
	if heightCDP != heightAcc {
		return fmt.Errorf("unmatched query height, cannot ensure no race condition")
	}
	// TODO ensure full node is not syncing

	if isWithinRange(augmentedCDP.CollateralizationRatio, app.lowerTrigger, app.upperTrigger) {
		return fmt.Errorf("collateral ratio (%s) has not deviated enough from target (%s)", augmentedCDP.CollateralizationRatio, app.targetRatio())
	}
	desiredDebtChange := calculateDebtAdjustment(
		augmentedCDP.CollateralizationRatio,
		totalPrinciple(augmentedCDP.CDP).Amount,
		app.targetRatio(),
	)
	if desiredDebtChange.Equal(sdk.ZeroInt()) {
		return fmt.Errorf("amount to rebalance is 0")
	}

	msg := app.constructMsg(desiredDebtChange, augmentedCDP.Principal.Denom)

	stdTx, err := constructSignedStdTx(app.signer, msg, account.GetAccountNumber(), account.GetSequence(), app.chainID, app.txFee)
	if err != nil {
		return fmt.Errorf("could not create tx: %w", err)
	}
	err = app.client.broadcastTx(stdTx)
	if err != nil {
		return fmt.Errorf("could not send tx: %w", err)
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
