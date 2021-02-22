package spammer

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/x/cdp"
	"github.com/kava-labs/kava/x/hard"

	"github.com/kava-labs/go-tools/spammer/client"
)

const (
	mnemonic                = "fragile flip puzzle adjust mushroom gas minimum maid love coach brush cattle match analyst oak spell blur thunder unfair inch mother park toilet toddler"
	rpcAddr                 = "tcp://localhost:26657"
	CreateCDPTxDefaultGas   = 500_000
	DepositHardTxDefaultGas = 200_000
	// TxConfirmationTimeout is the longest time to wait for a tx confirmation before giving up
	TxConfirmationTimeout      = 3 * 60 * time.Second
	TxConfirmationPollInterval = 2 * time.Second
)

var (
	DefaultGasPrice sdk.DecCoin = sdk.NewDecCoinFromDec("ukava", sdk.MustNewDecFromStr("0.25"))
)

type Spammer struct {
	client      *client.KavaClient
	distributor keys.KeyManager
	accounts    []keys.KeyManager
}

func NewSpammer(kavaClient *client.KavaClient, distributor keys.KeyManager, accounts []keys.KeyManager) Spammer {
	return Spammer{
		client:      kavaClient,
		distributor: distributor,
		accounts:    accounts,
	}
}

// DistributeCoins distributes coins from the spammer's distributor account to the general accounts
func (s Spammer) DistributeCoins(perAddrAmount int64) error {
	var inputs []bank.Input
	var outputs []bank.Output

	// Construct inputs
	totalDistCoins := sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(int64(len(s.accounts))*perAddrAmount)))

	// TODO: check that address has enough coins
	// if totalDistCoins.IsAllLT(senderAcc.Coins) {
	// 	return fmt.Errorf(fmt.Sprintf("sender %s has %s coins, needs %s"), s.client.Keybase.GetAddr(), senderAcc.Coins, totalDistCoins)
	// }

	input := bank.NewInput(s.distributor.GetAddr(), totalDistCoins)
	inputs = append(inputs, input)

	// Construct outputs
	perUserCoins := sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(perAddrAmount)))
	for _, account := range s.accounts {
		output := bank.NewOutput(account.GetAddr(), perUserCoins)
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
		Fee:           calculateFee(CreateCDPTxDefaultGas, DefaultGasPrice),
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
	res, err := s.client.BroadcastTxCommit(tx)
	if err != nil {
		return err
	}
	if res.CheckTx.Code != 0 {
		return fmt.Errorf("\nres.Code: %d\nLog:%s", res.CheckTx.Code, res.CheckTx.Log)
	}

	fmt.Println(fmt.Sprintf("Sent %s each to %d accounts: %s", perUserCoins, len(s.accounts), res.Hash))
	return nil
}

// OpenCDPs executes a series of CDP creations
func (s Spammer) OpenCDPs() error {
	collateralCoin := sdk.NewCoin("ukava", sdk.NewInt(10000000)) // 10 KAVA
	principleCoin := sdk.NewCoin("usdx", sdk.NewInt(10000000))   // 10 USDX
	collateralType := "ukava"

	fmt.Println(fmt.Sprintf("\nOpening CDPs with %s collateral, %s principal on each account...", collateralCoin, principleCoin))

	// Open CDPs
	for _, account := range s.accounts {
		fromAddr := account.GetAddr()

		msg := cdp.NewMsgCreateCDP(fromAddr, collateralCoin, principleCoin, collateralType)
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
			Fee:           calculateFee(CreateCDPTxDefaultGas, DefaultGasPrice),
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
		res, err := s.client.BroadcastTxSync(tx)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			return fmt.Errorf("\nres.Code: %d\nLog:%s", res.Code, res.Log)
		}
		fmt.Println(fmt.Sprintf("Sent tx %s, confirming...", res.Hash))

		err = pollWithBackoff(TxConfirmationTimeout, TxConfirmationPollInterval, func() (bool, error) {
			queryRes, err := s.client.GetTxConfirmation(res.Hash)
			if err != nil {
				return false, nil // poll again, it can't find the tx or node is down/slow
			}
			if queryRes.TxResult.Code != 0 {
				return true, fmt.Errorf("tx rejected from block: %s", queryRes.TxResult.Log) // return error, found tx but it didn't work
			}
			return true, nil // return nothing, found successfully confirmed tx
		})
	}
	fmt.Println(fmt.Sprintf("Successfully opened %d CDPs!", len(s.accounts)))
	return nil
}

// HardDeposits executes a series of Hard module deposits
func (s Spammer) HardDeposits() error {
	depositCoins := sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(10000000))) // 10 KAVA

	fmt.Println(fmt.Sprintf("\nSupplying %s to Hard on each account...", depositCoins))

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
		res, err := s.client.BroadcastTxSync(tx)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			return fmt.Errorf("\nres.Code: %d\nLog:%s", res.Code, res.Log)
		}
		fmt.Println(fmt.Sprintf("Sent tx %s, confirming...", res.Hash))

		err = pollWithBackoff(TxConfirmationTimeout, TxConfirmationPollInterval, func() (bool, error) {
			queryRes, err := s.client.GetTxConfirmation(res.Hash)
			if err != nil {
				return false, nil // poll again, it can't find the tx or node is down/slow
			}
			if queryRes.TxResult.Code != 0 {
				return true, fmt.Errorf("tx rejected from block: %s", queryRes.TxResult.Log) // return error, found tx but it didn't work
			}
			return true, nil // return nothing, found successfully confirmed tx
		})
	}
	fmt.Println(fmt.Sprintf("Successfully supplied on %d accounts!", len(s.accounts)))
	return nil
}

// HardBorrows executes a series of Hard module borrows
func (s Spammer) HardBorrows() error {
	depositCoins := sdk.NewCoins(sdk.NewCoin("usdx", sdk.NewInt(10000000))) // 10 USDX

	fmt.Println(fmt.Sprintf("\nBorrowing %s to Hard on each account...", depositCoins))

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
		res, err := s.client.BroadcastTxSync(tx)
		if err != nil {
			return err
		}
		if res.Code != 0 {
			return fmt.Errorf("\nres.Code: %d\nLog:%s", res.Code, res.Log)
		}
		fmt.Println(fmt.Sprintf("Sent tx %s, confirming...", res.Hash))

		err = pollWithBackoff(TxConfirmationTimeout, TxConfirmationPollInterval, func() (bool, error) {
			queryRes, err := s.client.GetTxConfirmation(res.Hash)
			if err != nil {
				return false, nil // poll again, it can't find the tx or node is down/slow
			}
			if queryRes.TxResult.Code != 0 {
				return true, fmt.Errorf("tx rejected from block: %s", queryRes.TxResult.Log) // return error, found tx but it didn't work
			}
			return true, nil // return nothing, found successfully confirmed tx
		})
	}
	fmt.Println(fmt.Sprintf("Successfully borrowed on %d accounts!", len(s.accounts)))
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

// pollWithBackoff will call the provided function until either:
// it returns true, it returns an error, the timeout passes.
// It will wait initialInterval after the first call, and double each subsequent call.
func pollWithBackoff(timeout, initialInterval time.Duration, pollFunc func() (bool, error)) error {
	const backoffMultiplier = 2
	deadline := time.After(timeout)

	wait := initialInterval
	nextPoll := time.After(0)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("polling timed out after %s", timeout)
		case <-nextPoll:
			shouldStop, err := pollFunc()
			if shouldStop || err != nil {
				return err
			}
			nextPoll = time.After(wait)
			wait = wait * backoffMultiplier
		}
	}
}
