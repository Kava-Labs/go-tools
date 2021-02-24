package spammer

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/prometheus/common/log"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/x/cdp"
	"github.com/kava-labs/kava/x/hard"

	"github.com/kava-labs/go-tools/spammer/client"
)

const (
	CreateCDPTxDefaultGas   = 500_000
	DepositHardTxDefaultGas = 200_000
	BorrowHardTxDefaultGas  = 200_000
	// TxConfirmationTimeout is the longest time to wait for a tx confirmation before giving up
	TxConfirmationTimeout      = 3 * 60 * time.Second
	TxConfirmationPollInterval = 2 * time.Second
)

var (
	DefaultGasPrice sdk.DecCoin = sdk.NewDecCoinFromDec("ukava", sdk.MustNewDecFromStr("0.05"))
)

// Spammer contains a Kava client, as well as Key Managers for a distribution account and a set of sub-accounts
type Spammer struct {
	client      *client.KavaClient
	distributor keys.KeyManager
	accounts    []keys.KeyManager
}

// NewSpammer returns a new instance of Spammer
func NewSpammer(kavaClient *client.KavaClient, distributor keys.KeyManager, accounts []keys.KeyManager) Spammer {
	return Spammer{
		client:      kavaClient,
		distributor: distributor,
		accounts:    accounts,
	}
}

// DistributeCoins distributes coins from the spammer's distributor account to the general accounts
func (s Spammer) DistributeCoins(perAddrAmount sdk.Coins) error {
	log.Infof("Distributing %s to each account...", perAddrAmount)

	var inputs []bank.Input
	var outputs []bank.Output

	// Construct inputs
	totalDistCoins := sdk.NewCoins()
	for _, coin := range perAddrAmount {
		newAmount := coin.Amount.Mul(sdk.NewInt(int64(len(s.accounts))))
		totalCoin := sdk.NewCoin(coin.Denom, newAmount)
		totalDistCoins = totalDistCoins.Add(totalCoin)
	}

	// TODO: check that address has enough coins
	// if totalDistCoins.IsAllLT(senderAcc.Coins) {
	// 	return fmt.Errorf(fmt.Sprintf("sender %s has %s coins, needs %s"), s.client.Keybase.GetAddr(), senderAcc.Coins, totalDistCoins)
	// }

	input := bank.NewInput(s.distributor.GetAddr(), totalDistCoins)
	inputs = append(inputs, input)

	// Construct outputs
	for _, account := range s.accounts {
		output := bank.NewOutput(account.GetAddr(), perAddrAmount)
		outputs = append(outputs, output)
	}

	// Construct MsgMultiSend
	msg := bank.NewMsgMultiSend(inputs, outputs)
	if err := msg.ValidateBasic(); err != nil {
		return fmt.Errorf("msg basic validation failed: \n%v", err)
	}

	chainID, err := s.client.GetChainID()
	if err != nil {
		return err
	}

	signMsg := &authtypes.StdSignMsg{
		ChainID:       chainID,
		AccountNumber: 0,
		Sequence:      0,
		Fee:           calculateFee(20000000, DefaultGasPrice),
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}

	sequence, accountNumber, err := getAccountNumbers(s.client, s.distributor.GetAddr())
	if err != nil {
		return err
	}
	signMsg.Sequence = sequence
	signMsg.AccountNumber = accountNumber

	signedMsg, err := s.distributor.Sign(*signMsg, s.client.Cdc)
	if err != nil {
		return err
	}
	tx := tmtypes.Tx(signedMsg)

	// Broadcast msg
	res, err := s.client.Http.BroadcastTxAsync(tx)
	if err != nil {
		return err
	}

	// TODO:
	// if res.CheckTx.Code != 0 {
	// 	return fmt.Errorf("\nres.Code: %d\nLog:%s", res.CheckTx.Code, res.CheckTx.Log)
	// }
	log.Infof("Sent tx %s", res.Hash)
	return nil
}

// OpenCDPs executes a series of CDP creations
func (s Spammer) OpenCDPs(collateralCoin, principalCoin sdk.Coin, collateralType string) error {

	log.Infof("\nOpening CDPs with %s collateral, %s principal on each account...", collateralCoin, principalCoin)

	// Open CDPs
	for _, account := range s.accounts {
		fromAddr := account.GetAddr()

		msg := cdp.NewMsgCreateCDP(fromAddr, collateralCoin, principalCoin, collateralType)
		if err := msg.ValidateBasic(); err != nil {
			return fmt.Errorf("msg basic validation failed: \n%v", err)
		}

		chainID, err := s.client.GetChainID()
		if err != nil {
			return err
		}

		signMsg := &authtypes.StdSignMsg{
			ChainID:       chainID,
			AccountNumber: 0,
			Sequence:      0,
			Fee:           calculateFee(500000, DefaultGasPrice),
			Msgs:          []sdk.Msg{msg},
			Memo:          "",
		}

		sequence, accountNumber, err := getAccountNumbers(s.client, fromAddr)
		if err != nil {
			return err
		}
		signMsg.Sequence = sequence
		signMsg.AccountNumber = accountNumber

		signedMsg, err := account.Sign(*signMsg, s.client.Cdc)
		if err != nil {
			return err
		}
		tx := tmtypes.Tx(signedMsg)

		maxTxLength := 1024 * 1024
		if len(tx) > maxTxLength {
			return fmt.Errorf("the tx data exceeds max length %d ", maxTxLength)
		}

		// Broadcast msg
		res, err := s.client.Http.BroadcastTxAsync(tx)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			return fmt.Errorf("\nres.Code: %d\nLog:%s", res.Code, res.Log)
		}
		log.Infof("Sent tx %s", res.Hash)
	}
	log.Infof("Successfully opened %d CDPs!", len(s.accounts))
	return nil
}

// HardDeposits executes a series of Hard module deposits
func (s Spammer) HardDeposits(depositCoins sdk.Coins) error {

	log.Infof("\nSupplying %s to Hard on each account...", depositCoins)

	// Open CDPs
	for _, account := range s.accounts {
		fromAddr := account.GetAddr()

		msg := hard.NewMsgDeposit(fromAddr, depositCoins)
		if err := msg.ValidateBasic(); err != nil {
			return fmt.Errorf("msg basic validation failed: \n%v", err)
		}

		chainID, err := s.client.GetChainID()
		if err != nil {
			return err
		}

		signMsg := &authtypes.StdSignMsg{
			ChainID:       chainID,
			AccountNumber: 0,
			Sequence:      0,
			Fee:           calculateFee(DepositHardTxDefaultGas, DefaultGasPrice),
			Msgs:          []sdk.Msg{msg},
			Memo:          "",
		}

		sequence, accountNumber, err := getAccountNumbers(s.client, fromAddr)
		if err != nil {
			return err
		}
		signMsg.Sequence = sequence
		signMsg.AccountNumber = accountNumber

		signedMsg, err := account.Sign(*signMsg, s.client.Cdc)
		if err != nil {
			return err
		}
		tx := tmtypes.Tx(signedMsg)

		maxTxLength := 1024 * 1024
		if len(tx) > maxTxLength {
			return fmt.Errorf("the tx data exceeds max length %d ", maxTxLength)
		}

		// Broadcast msg
		res, err := s.client.Http.BroadcastTxAsync(tx)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			return fmt.Errorf("\nres.Code: %d\nLog:%s", res.Code, res.Log)
		}
		log.Infof("Sent tx %s", res.Hash)
	}
	log.Infof("Successfully supplied on %d accounts!", len(s.accounts))
	return nil
}

// HardBorrows executes a series of Hard module borrows
func (s Spammer) HardBorrows(depositCoins sdk.Coins) error {

	log.Infof("\nBorrowing %s to Hard on each account...", depositCoins)

	// Open CDPs
	for _, account := range s.accounts {
		fromAddr := account.GetAddr()

		msg := hard.NewMsgBorrow(fromAddr, depositCoins)
		if err := msg.ValidateBasic(); err != nil {
			return fmt.Errorf("msg basic validation failed: \n%v", err)
		}

		chainID, err := s.client.GetChainID()
		if err != nil {
			return err
		}

		signMsg := &authtypes.StdSignMsg{
			ChainID:       chainID,
			AccountNumber: 0,
			Sequence:      0,
			Fee:           calculateFee(DepositHardTxDefaultGas, DefaultGasPrice),
			Msgs:          []sdk.Msg{msg},
			Memo:          "",
		}

		sequence, accountNumber, err := getAccountNumbers(s.client, fromAddr)
		if err != nil {
			return err
		}
		signMsg.Sequence = sequence
		signMsg.AccountNumber = accountNumber

		signedMsg, err := account.Sign(*signMsg, s.client.Cdc)
		if err != nil {
			return err
		}
		tx := tmtypes.Tx(signedMsg)

		maxTxLength := 1024 * 1024
		if len(tx) > maxTxLength {
			return fmt.Errorf("the tx data exceeds max length %d ", maxTxLength)
		}

		// Broadcast msg
		res, err := s.client.Http.BroadcastTxAsync(tx)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			return fmt.Errorf("\nres.Code: %d\nLog:%s", res.Code, res.Log)
		}
		log.Infof("Sent tx %s", res.Hash)
	}
	log.Infof("Successfully borrowed on %d accounts!", len(s.accounts))
	return nil
}

// calculateFee calculates the total fee to be paid based on a total gas and gas price.
func calculateFee(gas uint64, gasPrice sdk.DecCoin) authtypes.StdFee {
	var coins sdk.Coins
	if gas > 0 {
		coins = sdk.NewCoins(sdk.NewCoin(
			gasPrice.Denom,
			gasPrice.Amount.MulInt64(int64(gas)).Ceil().TruncateInt(),
		))
	}
	return authtypes.NewStdFee(gas, coins)
}

func getAccountNumbers(client *client.KavaClient, fromAddr sdk.AccAddress) (uint64, uint64, error) {
	acc, err := client.GetAccount(fromAddr)
	if err != nil {
		return 0, 0, err
	}
	return acc.GetSequence(), acc.GetAccountNumber(), nil
}
