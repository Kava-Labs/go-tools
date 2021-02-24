package main

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

func GetAccAddress(privKey tmcrypto.PrivKey) sdk.AccAddress {
	return privKey.PubKey().Address().Bytes()
}

func Sign(privKey tmcrypto.PrivKey, signMsg authtypes.StdSignMsg) (authtypes.StdTx, error) {
	sigBytes, err := privKey.Sign(signMsg.Bytes())
	if err != nil {
		return authtypes.StdTx{}, err
	}

	sig := authtypes.StdSignature{
		PubKey:    privKey.PubKey(),
		Signature: sigBytes,
	}

	tx := authtypes.NewStdTx(
		signMsg.Msgs,
		signMsg.Fee,
		[]authtypes.StdSignature{sig},
		signMsg.Memo,
	)

	return tx, nil
}
