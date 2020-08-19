// Package common isolates config for the kava and bnb nodes used in integration tests.
package common

import (
	"errors"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
)

const (
	KavaNodeURL  = "tcp://localhost:26657"
	KavaRestURL  = "http://localhost:1317"
	KavaChainID  = "testing"
	KavaGovDenom = "ukava"
)

var (
	// these are the same as the mnemonics in the chain config
	KavaOracleMnemonic = "law assault face proud fan slim genius boring portion delay team rude vapor timber noble absorb laugh dilemma patch actress brisk tissue drift flock"
	KavaDeputyMnemonic = "equip town gesture square tomorrow volume nephew minute witness beef rich gadget actress egg sing secret pole winter alarm law today check violin uncover"

	KavaUserMnemonics = []string{
		"very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food",
		"curtain camp spoil tiny vehicle pottery deer corn truly banner salmon lift yard throw open move state lamp van sign glow glue shrug faith",
		"desert october mammal tuition illness album engine solid enjoy harvest symptom rely camera unable okay avocado actual oppose remember lady dove canal argue cave",
		"profit law bounce grunt earth ice share skill valve awful around shoot include kite lecture also smooth ball vintage snake embark brief ill gather",
		"census museum crew rude tower vapor mule rib weasel faith page cushion rain inherit much cram that blanket occur region track hub zero topple",
		"flavor print loyal canyon expand salmon century field say frequent human dinosaur frame claim bridge affair web way direct win become merry crash frequent",
	}

	KavaOracleAddr sdk.AccAddress
	KavaDeputyAddr sdk.AccAddress
	KavaUserAddrs  []sdk.AccAddress
)

func init() {
	// set the prefix
	// note: this will set the prefix for any package that imports this package
	kavaConfig := sdk.GetConfig()
	app.SetBech32AddressPrefixes(kavaConfig)
	app.SetBip44CoinType(kavaConfig)

	KavaOracleAddr = KavaAddressFromMnemonic(KavaOracleMnemonic)
	KavaDeputyAddr = KavaAddressFromMnemonic(KavaDeputyMnemonic)
	for _, m := range KavaUserMnemonics {
		KavaUserAddrs = append(KavaUserAddrs, KavaAddressFromMnemonic(m))
	}
}

// Wait will poll the provided function until either:
// - it returns true
// - it returns an error
// - the timeout passes
func Wait(timeout time.Duration, shouldStop func() (bool, error)) error {
	endTime := time.Now().Add(timeout)

	for {
		stop, err := shouldStop()
		switch {
		case err != nil || stop:
			return err
		case time.Now().After(endTime):
			return errors.New("waiting timed out")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// KavaAddressFromMnemonic converts a mnemonic to a kava address, using the default bip44 path.
func KavaAddressFromMnemonic(mnemonic string) sdk.AccAddress {
	keybase := keys.NewInMemory()
	bip39Password := ""
	encryptPassword := ""
	hdPath := keys.CreateHDPath(0, 0).String()
	info, err := keybase.CreateAccount("integration-test-setup", mnemonic, bip39Password, encryptPassword, hdPath, keys.Secp256k1)
	if err != nil {
		panic(err)
	}
	return info.GetAddress()
}
