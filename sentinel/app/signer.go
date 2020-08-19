package app

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type TxSigner struct {
	keybase            keys.Keybase
	keybaseKeyName     string
	keybaseKeyPassword string
}

func NewDefaultTxSigner(mnemonic string) (TxSigner, error) {
	hdPath := keys.CreateHDPath(0, 0).String()
	return NewTxSigner(mnemonic, hdPath, keys.Secp256k1)
}
func NewTxSigner(mnemonic, hdPath string, signingAlgo keys.SigningAlgo) (TxSigner, error) {
	keybase := keys.NewInMemory()
	bip39Password := ""
	keyPassword := "password"
	keyName := "key-name"
	_, err := keybase.CreateAccount(keyName, mnemonic, bip39Password, keyPassword, hdPath, signingAlgo)
	if err != nil {
		return TxSigner{}, err
	}
	return TxSigner{
		keybase:            keybase,
		keybaseKeyName:     keyName,
		keybaseKeyPassword: keyPassword,
	}, nil
}
func (txSigner TxSigner) Sign(stdTx authtypes.StdTx, accountNumber, sequence uint64, chainID string) (authtypes.StdSignature, error) {
	// find the raw bytes to sign
	signBytes := authtypes.StdSignBytes(chainID, accountNumber, sequence, stdTx.Fee, stdTx.Msgs, stdTx.Memo)

	// create raw signature
	signatureBytes, pubKey, err := txSigner.keybase.Sign(txSigner.keybaseKeyName, txSigner.keybaseKeyPassword, signBytes)
	if err != nil {
		return authtypes.StdSignature{}, err
	}
	// return signature in standard format
	return authtypes.StdSignature{
		PubKey:    pubKey,
		Signature: signatureBytes,
	}, nil
}

func (txSigner TxSigner) GetAddress() sdk.AccAddress {
	info, err := txSigner.keybase.Get(txSigner.keybaseKeyName)
	if err != nil {
		panic(err) // an in memory db should not fail to retreive item
	}
	return info.GetAddress()
}
