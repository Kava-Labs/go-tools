package app

import sdk "github.com/cosmos/cosmos-sdk/types"

type App struct {
	client   Client
	cdpOwner sdk.AccAddress
	cdpDenom string
}

func NewApp(restURL string, cdpOwner sdk.AccAddress, cdpDenom string) App {
	return App{
		client:   NewClient(restURL),
		cdpOwner: cdpOwner,
		cdpDenom: cdpDenom,
	}
}
func (app App) Run() error {
	// cdp, err := app.client.getCDP(app.cdpOwner, app.cdpDenom)
	// if err != nil {
	// 	return err
	// }
	// account, err := app.client.getAccount(app.cdpOwner)
	// if err != nil {
	// 	return err
	// }
	// // TODO get price
	// // TODO check heights match, if not start again

	// desiredDebt := cdp.Collateral.Mul(price).Div(desiredRatio)
	// desiredDebtChange := desiredDebt.Sub(cdp.GetTotalDebt())
}
