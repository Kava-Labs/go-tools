// Package addresses defines kava and bnb chain addresses used for integration testing.
package addresses

import (
	"encoding/json"
	"errors"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	kavaKeys "github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/app"
	"gopkg.in/yaml.v3"
)

const (
	KavaNodeURL                                           = "tcp://localhost:26657"
	KavaRestURL                                           = "http://localhost:1317"
	BnbNodeURL                                            = "tcp://localhost:26658"
	BnbHTLTFee, BnbClaimFee, BnbRefundFee, BnbTransferFee = 37500, 37500, 37500, 37500
)

const addresses = `
kava:
  validators:
    - mnemonic: "very health column only surface project output absent outdoor siren reject era legend legal twelve setup roast lion rare tunnel devote style random food"
      address: "kava1ypjp0m04pyp73hwgtc0dgkx0e9rrydecm054da"
      val_address: "kavavaloper1ypjp0m04pyp73hwgtc0dgkx0e9rrydeckewa42"
      cons_pubkey: "kavavalconspub1zcjduepqvfq6egzgfmdkd6k7cqhsvsfr4lhsp6adh4uurxgkhec8h7amxcjq7gjum4"
  deputys:
    bnb:
      hot_wallet:
        mnemonic: "curtain camp spoil tiny vehicle pottery deer corn truly banner salmon lift yard throw open move state lamp van sign glow glue shrug faith"
        address: "kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm"
      cold_wallet:
        mnemonic: "profit law bounce grunt earth ice share skill valve awful around shoot include kite lecture also smooth ball vintage snake embark brief ill gather"
        address: "kava1g33w0mh4mjllhaj3y4dcwkwquxgwrma9ga5t94"
    btcb:
      hot_wallet:
        mnemonic: "shed crush identify inmate fault truck raw sausage afford fiction day delay people shrimp firm group maple square host thank motor radio visual cable"
        address: "kava1kla4wl0ccv7u85cemvs3y987hqk0afcv7vue84"
      cold_wallet:
        mnemonic: "dirt better attack pulse amused derive female top wink cycle surge tell leopard remove nephew retreat allow stuff pipe come dinner globe open usage"
        address: "kava1ynf22ap74j6znl503a56y23x5stfr0aw5kntp8"
    xrpb:
      hot_wallet:
        mnemonic: "trial friend silly sugar maid behave slim onion swap report inmate hold hammer hip wrist above sketch mean fence reason master green panel chimney"
        address: "kava14q5sawxdxtpap5x5sgzj7v4sp3ucncjlpuk3hs"
      cold_wallet:
        mnemonic: "stumble output prefer toast trip earth turn husband present dad fashion lizard hero protect blood expand book parrot ahead ensure alien shiver twice pledge"
        address: "kava1z3ytjpr6ancl8gw80z6f47z9smug7986x29vtj"
    busd:
      hot_wallet:
        mnemonic: "grab charge flame lamp genuine accuse truth orange split can faith spoon twist romance input raccoon tissue slice hire sauce hope fork primary unlock"
        address: "kava1j9je7f6s0v6k7dmgv6u5k5ru202f5ffsc7af04"
      cold_wallet:
        mnemonic: "arrive guide way exit polar print kitchen hair series custom siege afraid shrug crew fashion mind script divorce pattern trust project regular robust safe"
        address: "kava1ektgdyy0z23qqnd67ns3qvfzgfgjd5xe82lf5c"
  oracles:
    - mnemonic: "desert october mammal tuition illness album engine solid enjoy harvest symptom rely camera unable okay avocado actual oppose remember lady dove canal argue cave"
      address: "kava1acge4tcvhf3q6fh53fgwaa7vsq40wvx6wn50em"
  committee_members:
    - mnemonic: "grass luxury welcome dismiss legal nothing glide crisp material broccoli jewel put inflict expose taxi wear second party air hockey crew ride wage nurse"
      address: "kava1n96qpdfcz2m7y364ewk8srv9zuq6ucwduyjaag"
  users:
    - mnemonic: "season bone lucky dog depth pond royal decide unknown device fruit inch clock trap relief horse morning taxi bird session throw skull avocado private"
      address: "kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc"
    - mnemonic: "twice brief orbit assist average victory shrimp visit rookie nation sentence obscure all deny immense borrow debate demise gorilla fault session transfer wide because"
      address: "kava1fwfwmt6vupf3m9uvpdsuuc4dga8p5dtl4npcqz"
    - mnemonic: "census museum crew rude tower vapor mule rib weasel faith page cushion rain inherit much cram that blanket occur region track hub zero topple"
      address: "kava1sw54s6gq76acm35ls6m5c0kr93dstgrh6eftld"
    - mnemonic: "flavor print loyal canyon expand salmon century field say frequent human dinosaur frame claim bridge affair web way direct win become merry crash frequent"
      address: "kava1t4dvu32e309pzhmdn3aqcjlj79h9876plynrfm"
    - mnemonic: "height space double mask panic fashion soon bright rude narrow shuffle pull scale box science plastic plug churn hub donor reason piece learn police"
      address: "kava1wuzhkn2f8nqe2aprnwt3jkjvvr9m7dlkpumtz2"
bnb:
  validators:
    - mnemonic: "village fiscal december liquid better drink disorder unusual tent ivory cage diesel bike slab tilt spray wife neck oak science beef upper chapter blade"
      address: "bnb1gg3a488t69ueg79lqneqx0h3442x7gtu8drvjz"
      val_address: "bva1gg3a488t69ueg79lqneqx0h3442x7gtu83zuvx"
      cons_pubkey: "bcap1zcjduepqk8p92cnpwgl22ly733hpdttzfyhy9ksxppv40jyjtuf2zzkpqz6qn9phhh"
  deputys:
    btcb:
      hot_wallet:
        mnemonic: "enjoy soldier replace ugly glimpse rude sponsor hood jewel inner hole tower initial drive jungle resist answer display capable give lesson mule spray whisper"
        address: "bnb1z8ryd66lhc4d9c0mmxx9zyyq4t3cqht9mt0qz3"
      cold_wallet:
        mnemonic: "cancel library license destroy regret confirm fancy cupboard crew elite staff noodle slogan identify eternal tone gown pair donor list present forget rain awful"
        address: "bnb1tpgqfslnm486qtnfatewdeya2khaav3x6hqhf9"
    bnb:
      hot_wallet:
        mnemonic: "almost design doctor exist destroy candy zebra insane client grocery govern idea library degree two rebuild coffee hat scene deal average fresh measure potato"
        address: "bnb1zfa5vmsme2v3ttvqecfleeh2xtz5zghh49hfqe"
      cold_wallet:
        mnemonic: "welcome bean crystal pave chapter process bless tribe inside bottom exhaust hollow display envelope rally moral admit round hidden junk silly afraid awesome muffin"
        address: "bnb1nva9yljftdf6m2dwhufk5kzg204jg060sw0fv2"
    xrpb:
      hot_wallet:
        mnemonic: "forward argue march dignity puzzle celery caught maze judge chair cement choice bamboo pulse else local foam abuse crazy bullet feed hero rose seat"
        address: "bnb1ryrenacljwghhc5zlnxs3pd86amta3jcaagyt0"
      cold_wallet:
        mnemonic: "exile theory key tree range grow shy pave round roof thing grow audit between echo join split certain gaze culture slab hour fury wool"
        address: "bnb13l6w5vzqa3533ukrpzpycqh62qd4r3tman8cnv"
    busd:
      hot_wallet:
        mnemonic: "bachelor also save receive tennis equal sign frog purse elevator gesture elegant legend drastic sorry sing consider project decrease critic thought screen detect honey"
        address: "bnb1j20j0e62n2l9sefxnu596a6jyn5x29lk2syd5j"
      cold_wallet:
        mnemonic: "canoe all morning buffalo wet sting truck rebuild section raven rent lecture model pink burger alone bachelor amazing similar soon pretty doll youth invite"
        address: "bnb1q8d7pl0546qa08cc4ptatmfzwfs47ztlsgsfqz"
  users:
    - mnemonic: "smile air crush cart puppy until upon distance pretty cabbage insect dream bargain more lift urban armor source case judge process cute seed verb"
      address: "bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x"
    - mnemonic: "cloud deal hurdle sound scout merit carpet identify fossil brass ancient keep disorder save lobster whisper course intact winter bullet flame mother upgrade install"
      address: "bnb15udkjukldcejs3y3pep2jvwza0lfzwshv7ndfr"
    - mnemonic: "certain bright injury decorate own scale harbor fiscal skirt violin shove clever prevent relax novel impulse gasp fold produce arrow push guide bunker kiss"
      address: "bnb1fzrs9etlhvg3k4wcrr6zqh4vdslld7933ln5a7"
    - mnemonic: "mutual duck begin remind release brave patrol squeeze abandon pact valid close fragile plastic disorder saddle bring inspire corn kitten reduce candy side honey"
      address: "bnb1gzr37hrqlqk4wpdfqn6pp3dy9ek28tahy7dx2v"
    - mnemonic: "end bicycle walnut empty bus silly camera lift fancy symptom office pluck detail unable cry sense scrap tuition relax amateur hold win debate hat"
      address: "bnb1m5k0g7q0n8nzsre8ysuhsv09j03jhgcrrnwur3"
`

type Addresses struct {
	Kava struct {
		Deputys struct {
			Bnb struct {
				HotWallet struct {
					Address  sdk.AccAddress `json:"address"`
					Mnemonic string         `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"bnb"`
			Btcb struct {
				HotWallet struct {
					Address  sdk.AccAddress `json:"address"`
					Mnemonic string         `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"btcb"`
			Busd struct {
				HotWallet struct {
					Address  sdk.AccAddress `json:"address"`
					Mnemonic string         `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"busd"`
			Xrpb struct {
				HotWallet struct {
					Address  sdk.AccAddress `json:"address"`
					Mnemonic string         `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"xrpb"`
		} `json:"deputys"`
		Users []struct {
			Address  sdk.AccAddress `json:"address"`
			Mnemonic string         `json:"mnemonic"`
		} `json:"users"`
	} `json:"kava"`
	Bnb struct {
		Deputys struct {
			Bnb struct {
				HotWallet struct {
					Address  types.AccAddress `json:"address"`
					Mnemonic string           `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"bnb"`
			Btcb struct {
				HotWallet struct {
					Address  types.AccAddress `json:"address"`
					Mnemonic string           `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"btcb"`
			Busd struct {
				HotWallet struct {
					Address  types.AccAddress `json:"address"`
					Mnemonic string           `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"busd"`
			Xrpb struct {
				HotWallet struct {
					Address  types.AccAddress `json:"address"`
					Mnemonic string           `json:"mnemonic"`
				} `json:"hot_wallet"`
			} `json:"xrpb"`
		} `json:"deputys"`
		Users []struct {
			Address  types.AccAddress `json:"address"`
			Mnemonic string           `json:"mnemonic"`
		} `json:"users"`
	} `json:"bnb"`
}

func GetAddresses() Addresses {
	// Unmarshal the yaml string constant into the Addresses type.
	// v0.39.1 of the cosmos-sdk has a bug where the sdk.AccAddress type doesn't define a UnmarshalYAML method correctly, so we can't unmarshal directly to the Addresses type.
	// Work around this by routing through json to pick up the correctly defined UnmarshalJSON methods.

	unmarshalStructure := map[string]interface{}{}
	err := yaml.Unmarshal([]byte(addresses), &unmarshalStructure)
	if err != nil {
		panic(err)
	}
	bz, err := json.Marshal(unmarshalStructure)
	if err != nil {
		panic(err)
	}
	var addresses Addresses
	if err := json.Unmarshal(bz, &addresses); err != nil {
		panic(err)
	}
	return addresses
}

func (addrs Addresses) KavaUserMnemonics() []string {
	var mnemonics []string
	for _, w := range addrs.Kava.Users {
		mnemonics = append(mnemonics, w.Mnemonic)
	}
	return mnemonics
}
func (addrs Addresses) BnbUserMnemonics() []string {
	var mnemonics []string
	for _, w := range addrs.Bnb.Users {
		mnemonics = append(mnemonics, w.Mnemonic)
	}
	return mnemonics
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
		time.Sleep(500 * time.Millisecond)
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
	manager, err := kavaKeys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
	if err != nil {
		panic(err.Error())
	}

	return manager.GetKeyRing().GetAddress()
}
