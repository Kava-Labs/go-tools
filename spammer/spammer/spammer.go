package spammer

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/go-bip39"
	log "github.com/sirupsen/logrus"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/app"
	"github.com/kava-labs/kava/x/cdp"
	"github.com/kava-labs/kava/x/hard"

	"github.com/kava-labs/go-tools/spammer/client"
	"github.com/kava-labs/go-tools/spammer/types"
)

const (
	CreateCDPTxDefaultGas = 500_000
	MultisendTxGas        = 250_000
	TxDefaultGas          = 200_000
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
func NewSpammer(kavaClient *client.KavaClient, distributor keys.KeyManager, numAccs int) (Spammer, error) {
	accounts, err := genNewAccounts(numAccs)
	if err != nil {
		return Spammer{}, err
	}

	spammer := Spammer{
		client:      kavaClient,
		distributor: distributor,
		accounts:    accounts,
	}
	return spammer, nil
}

// ProcessMsg processes a spammer message consisting of an sdk.Msg and processing details
func (s Spammer) ProcessMsg(message types.Message) error {
	if message.Processor.FromPrimaryAccount {
		err := s.processMsgViaPrimaryAccount(message)
		if err != nil {
			return err
		}
		return nil
	}

	err := s.processMsgViaSubAccounts(message)
	if err != nil {
		return err
	}
	return nil
}

func (s Spammer) processMsgViaPrimaryAccount(message types.Message) error {
	var msg sdk.Msg
	switch message.Msg.Type() {
	case "multisend": // Multisend loads all accounts
		msgMultisend, ok := message.Msg.(bank.MsgMultiSend)
		if !ok {
			return fmt.Errorf("invalid message structure: %s", message.Msg.Type())
		}

		if len(msgMultisend.Inputs) != 1 {
			return fmt.Errorf("multisend msg must only have one input")
		}
		msgMultisend.Inputs[0].Address = s.distributor.GetAddr()

		if len(s.accounts) == 0 {
			return fmt.Errorf("spammer requires at least one account to distribute to")
		}

		// Calculate coins per address
		perAddrCoins := sdk.NewCoins()
		for _, coin := range msgMultisend.Inputs[0].Coins {
			addrAmt := coin.Amount.Quo(sdk.NewInt(int64(len(s.accounts))))
			addrCoin := sdk.NewCoin(coin.Denom, addrAmt)
			perAddrCoins = perAddrCoins.Add(addrCoin)
		}

		// Create outputs for each account
		var outputs []bank.Output
		for _, account := range s.accounts {
			output := bank.NewOutput(account.GetAddr(), perAddrCoins)
			outputs = append(outputs, output)
		}
		msgMultisend.Outputs = outputs

		msg = msgMultisend
	default:
		return fmt.Errorf("unsupported message type: %s", message.Msg.Type())
	}

	err := s.broadcastMsg(msg, s.distributor)
	if err != nil {
		return err
	}
	return nil
}

func (s Spammer) processMsgViaSubAccounts(message types.Message) error {
	i := 0
	for i < message.Processor.Count {
		account := s.accounts[i]
		var msg sdk.Msg
		switch message.Msg.Type() {
		case "create_cdp":
			msgCreateCdp, ok := message.Msg.(cdp.MsgCreateCDP)
			if !ok {
				return fmt.Errorf("invalid message structure: %s", message.Msg.Type())
			}
			msgCreateCdp.Sender = account.GetAddr()
			msg = msgCreateCdp
		case "hard_deposit":
			msgHardDeposit, ok := message.Msg.(hard.MsgDeposit)
			if !ok {
				return fmt.Errorf("invalid message structure: %s", message.Msg.Type())
			}
			msgHardDeposit.Depositor = account.GetAddr()
			msg = msgHardDeposit
		case "hard_withdraw":
			msgHardWithdraw, ok := message.Msg.(hard.MsgWithdraw)
			if !ok {
				return fmt.Errorf("invalid message structure: %s", message.Msg.Type())
			}
			msgHardWithdraw.Depositor = account.GetAddr()
			msg = msgHardWithdraw
		default:
			return fmt.Errorf("unsupported message type: %s", message.Msg.Type())
		}

		err := s.broadcastMsg(msg, account)
		if err != nil {
			return err
		}
		i++
	}
	return nil
}

func (s Spammer) broadcastMsg(msg sdk.Msg, account keys.KeyManager) error {
	if err := msg.ValidateBasic(); err != nil {
		return fmt.Errorf("msg basic validation failed: \n%v", err)
	}

	chainID, err := s.client.GetChainID()
	if err != nil {
		return err
	}

	var fee authtypes.StdFee
	switch msg.Type() {
	case "multisend":
		multisendTxGas := uint64(MultisendTxGas + (len(s.accounts) * 20000)) // Scale gas with num accounts
		fee = calculateFee(multisendTxGas, DefaultGasPrice)
	case "create_cdp":
		fee = calculateFee(CreateCDPTxDefaultGas, DefaultGasPrice)
	default:
		fee = calculateFee(TxDefaultGas, DefaultGasPrice)
	}

	signMsg := &authtypes.StdSignMsg{
		ChainID:       chainID,
		AccountNumber: 0,
		Sequence:      0,
		Fee:           fee,
		Msgs:          []sdk.Msg{msg},
		Memo:          "",
	}

	sequence, accountNumber, err := getAccountNumbers(s.client, account.GetAddr())
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
	log.Infof("\tSent tx %s", res.Hash)

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

func genNewAccounts(count int) ([]keys.KeyManager, error) {
	var kavaKeys []keys.KeyManager
	for i := 0; i < count; i++ {
		entropySeed, err := bip39.NewEntropy(256)
		if err != nil {
			return kavaKeys, err
		}

		mnemonic, err := bip39.NewMnemonic(entropySeed)
		if err != nil {
			return kavaKeys, err
		}

		keyManager, err := keys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
		if err != nil {
			return kavaKeys, err
		}
		kavaKeys = append(kavaKeys, keyManager)
	}

	return kavaKeys, nil
}
