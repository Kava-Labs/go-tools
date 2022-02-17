package testutil

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client/flags"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/kava-labs/kava/app"

	"github.com/kava-labs/go-tools/signing"
)

func DefaultKavaAppGenesis(consensusKey cryptotypes.PubKey, valOperPrivKey cryptotypes.PrivKey, chainID string) app.GenesisState {
	govDenom := "ukava"

	encodingConfig := app.MakeEncodingConfig()
	cdc := encodingConfig.Marshaler

	genState := app.ModuleBasics.DefaultGenesis(cdc)

	// Setup account and balance for validator

	abBuilder := app.NewAuthBankGenesisBuilder().WithSimpleAccount(
		valOperPrivKey.PubKey().Address().Bytes(),
		sdk.NewCoins(sdk.NewInt64Coin(govDenom, 1e12)),
	)
	for k, v := range abBuilder.BuildMarshalled(cdc) {
		genState[k] = v
	}

	// Setup crisis, gov, mint, staking with kava denom

	stakingGen := stakingtypes.DefaultGenesisState()
	stakingGen.Params.BondDenom = govDenom
	genState[stakingtypes.ModuleName] = cdc.MustMarshalJSON(stakingGen)

	govGen := govtypes.DefaultGenesisState()
	govGen.DepositParams.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin(govDenom, 1e7))
	genState[govtypes.ModuleName] = cdc.MustMarshalJSON(govGen)

	mintGen := minttypes.DefaultGenesisState()
	mintGen.Params.MintDenom = govDenom
	genState[minttypes.ModuleName] = cdc.MustMarshalJSON(mintGen)

	crisisGen := crisistypes.DefaultGenesisState()
	crisisGen.ConstantFee = sdk.NewInt64Coin(govDenom, 1e3)
	genState[crisistypes.ModuleName] = cdc.MustMarshalJSON(crisisGen)

	// Add gentx

	valAddr := sdk.ValAddress(valOperPrivKey.PubKey().Address().Bytes())
	msg := DefaultCreateValidatorMsg(valAddr, consensusKey, sdk.NewInt64Coin(govDenom, 1e12))

	txBuilder := encodingConfig.TxConfig.NewTxBuilder()
	txBuilder.SetMsgs(msg)
	txBuilder.SetGasLimit(flags.DefaultGasLimit)

	tx, _, err := signing.Sign(
		encodingConfig.TxConfig,
		valOperPrivKey,
		txBuilder,
		authsigning.SignerData{
			ChainID:       chainID,
			AccountNumber: 0, // account and sequence 0 as only validator account is created above
			Sequence:      0,
		},
	)
	if err != nil {
		panic(err)
	}

	txJsonBytes, err := encodingConfig.TxConfig.TxJSONEncoder()(tx)
	if err != nil {
		panic(err)
	}

	genutilGen := genutiltypes.DefaultGenesisState()
	genutilGen.GenTxs = []json.RawMessage{txJsonBytes}
	genState[genutiltypes.ModuleName] = cdc.MustMarshalJSON(genutilGen)

	return genState
}

func DefaultCreateValidatorMsg(address sdk.ValAddress, pubKey cryptotypes.PubKey, delegation sdk.Coin) *stakingtypes.MsgCreateValidator {
	msg, err := stakingtypes.NewMsgCreateValidator(
		address,
		pubKey,
		delegation,
		stakingtypes.Description{
			Moniker: address.String(), // Description cannot be empty
		},
		stakingtypes.CommissionRates{ // defaults taken from genutil
			Rate:          sdk.MustNewDecFromStr("0.1"),
			MaxRate:       sdk.MustNewDecFromStr("0.2"),
			MaxChangeRate: sdk.MustNewDecFromStr("0.01"),
		},
		sdk.NewInt(1e6), // minimum allowed by sdk
	)
	if err != nil {
		panic(err)
	}
	return msg
}
