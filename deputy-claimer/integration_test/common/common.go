// Package common isolates config for the kava and bnb nodes used in integration tests.
package common

import (
	"errors"
	"fmt"
	"time"

	"github.com/binance-chain/go-sdk/common/types"
	bnbKeys "github.com/binance-chain/go-sdk/keys"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/kava"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
)

const (
	BnbHTLTFee, BnbClaimFee, BnbRefundFee, BnbTransferFee = 37500, 37500, 37500, 37500
	BnbNodeURL                                            = "tcp://localhost:26658"
	KavaNodeURL                                           = "tcp://localhost:26657"
)

var (
	// these are the same as the mnemonics in the chains and deputy configs
	BnbDeputyMnemonic  = "clinic soap symptom alter mango orient punch table seek among broken bundle best dune hurt predict liquid subject silver once kick metal okay moment"
	KavaDeputyMnemonic = "equip town gesture square tomorrow volume nephew minute witness beef rich gadget actress egg sing secret pole winter alarm law today check violin uncover"

	KavaUserMnemonics = []string{
		"very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food",
		"curtain camp spoil tiny vehicle pottery deer corn truly banner salmon lift yard throw open move state lamp van sign glow glue shrug faith",
		"desert october mammal tuition illness album engine solid enjoy harvest symptom rely camera unable okay avocado actual oppose remember lady dove canal argue cave",
		"profit law bounce grunt earth ice share skill valve awful around shoot include kite lecture also smooth ball vintage snake embark brief ill gather",
		"census museum crew rude tower vapor mule rib weasel faith page cushion rain inherit much cram that blanket occur region track hub zero topple",
		"flavor print loyal canyon expand salmon century field say frequent human dinosaur frame claim bridge affair web way direct win become merry crash frequent",
	}
	BnbUserMnemonics = []string{
		"then nuclear favorite advance plate glare shallow enhance replace embody list dose quick scale service sentence hover announce advance nephew phrase order useful this",
		"almost design doctor exist destroy candy zebra insane client grocery govern idea library degree two rebuild coffee hat scene deal average fresh measure potato",
		"welcome bean crystal pave chapter process bless tribe inside bottom exhaust hollow display envelope rally moral admit round hidden junk silly afraid awesome muffin",
		"end bicycle walnut empty bus silly camera lift fancy symptom office pluck detail unable cry sense scrap tuition relax amateur hold win debate hat",
		"cloud deal hurdle sound scout merit carpet identify fossil brass ancient keep disorder save lobster whisper course intact winter bullet flame mother upgrade install",
		"mutual duck begin remind release brave patrol squeeze abandon pact valid close fragile plastic disorder saddle bring inspire corn kitten reduce candy side honey",
	}

	BnbDeputyAddr  types.AccAddress
	KavaDeputyAddr sdk.AccAddress
	BnbUserAddrs   []types.AccAddress
	KavaUserAddrs  []sdk.AccAddress
)

func init() {
	// set the prefix
	// note: this will set the prefix for any package that imports this package
	kavaConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(kavaConfig)

	BnbDeputyAddr = BnbAddressFromMnemonic(BnbDeputyMnemonic)
	fmt.Println("bnb dep", BnbDeputyAddr.String())
	for _, m := range BnbUserMnemonics {
		BnbUserAddrs = append(BnbUserAddrs, BnbAddressFromMnemonic(m))
	}
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

// BnbAddressFromMnemonic converts a mnemonic to a bnb address, using the default bip44 path.
func BnbAddressFromMnemonic(mnemonic string) types.AccAddress {
	manager, err := bnbKeys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(err.Error())
	}
	return manager.GetAddr()
}

// KavaAddressFromMnemonic converts a mnemonic to a kava address, using the default bip44 path.
func KavaAddressFromMnemonic(mnemonic string) sdk.AccAddress {
	manager, err := kavaKeys.NewMnemonicKeyManager(mnemonic, kava.Bip44CoinType)
	if err != nil {
		panic(err.Error())
	}
	return manager.GetAddr()
}
