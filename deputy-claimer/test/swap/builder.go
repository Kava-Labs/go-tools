package swap

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
)

var (
	denomMap = map[string]string{
		"XRP-BF2":  "xrpb",
		"BUSD-BD1": "busd",
		"BTCB-1DE": "btcb",
		"BNB":      "bnb",
	}
	deterministicRand          = rand.New(rand.NewSource(1234))
	defaultKavaToBnbHeightSpan = HeightSpan{
		Kava: 220, // must be > other_chain_min_accept_expire_height_span deputy param
		Bnb:  360, // bnb_expire_height_span param in deputy
	}
	defaultBnbToKavaHeightSpan = HeightSpan{
		Bnb:  10000, // must be > bnb_min_accept_expire_height_span deputy param
		Kava: 120,   // other_chain_expire_height_span deputy param
	}
)

// Builder assists in creating cross chain swaps by storing common swap parameters.
type Builder struct {
	kavaDeputyMnemonic  string
	bnbDeputyMnemonic   string
	calculateKavaAmount func(types.Coins) sdk.Coins
	calculateBnbAmount  func(sdk.Coins) types.Coins
	heightSpanKavaToBnb HeightSpan
	heightSpanBnbToKava HeightSpan
	genTimestamp        func() int64
	genRandomNumber     func() []byte
}

// NewDefaultBuilder creates a Builder with defaults for common swap parameters.
func NewDefaultBuilder(kavaDeputyMnemonic, bnbDeputyMnemonic string) Builder {
	return Builder{
		kavaDeputyMnemonic:  kavaDeputyMnemonic,
		bnbDeputyMnemonic:   bnbDeputyMnemonic,
		calculateKavaAmount: convertBnbToKavaCoins,
		calculateBnbAmount:  convertKavaToBnbCoins,
		heightSpanKavaToBnb: defaultKavaToBnbHeightSpan,
		heightSpanBnbToKava: defaultBnbToKavaHeightSpan,
		genTimestamp:        getCurrentTimestamp,
		genRandomNumber:     getDeterministicRandomNumber,
	}
}

// WithTimestamp returns a SwapBuilder with a fixed value for swap timestamps.
func (builder Builder) WithTimestamp(timestamp int64) Builder {
	builder.genTimestamp = func() int64 { return timestamp }
	return builder
}

// WithHeightSpans returns a SwapBuilder with different height_span values used for creating swaps.
func (builder Builder) WithHeightSpans(kavaToBnb, bnbToKava HeightSpan) Builder {
	builder.heightSpanBnbToKava = bnbToKava
	builder.heightSpanKavaToBnb = kavaToBnb
	return builder
}

// NewBnbToKavaSwap creates a cross chain swap using common parameters from the builder.
func (builder Builder) NewBnbToKavaSwap(senderMnemonic string, recipient sdk.AccAddress, amount types.Coins) CrossChainSwap {
	rndNum := builder.genRandomNumber()
	timestamp := builder.genTimestamp()
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
	return NewBnbToKavaSwap(
		senderMnemonic,
		recipient,
		Amount{
			Bnb:  amount,
			Kava: builder.calculateKavaAmount(amount),
		},
		builder.kavaDeputyMnemonic,
		bnbAddressFromMnemonic(builder.bnbDeputyMnemonic),
		rndHash,
		timestamp,
		builder.heightSpanBnbToKava,
		rndNum,
	)
}

// NewKavaToBnbSwap creates a cross chain swap using common parameters from the builder.
func (builder Builder) NewKavaToBnbSwap(senderMnemonic string, recipient types.AccAddress, amount sdk.Coins) CrossChainSwap {
	rndNum := builder.genRandomNumber()
	timestamp := builder.genTimestamp()
	rndHash := bep3types.CalculateRandomHash(rndNum, timestamp)
	return NewKavaToBnbSwap(
		senderMnemonic,
		recipient,
		Amount{
			Kava: amount,
			Bnb:  builder.calculateBnbAmount(amount),
		},
		builder.bnbDeputyMnemonic,
		kavaAddressFromMnemonic(builder.kavaDeputyMnemonic),
		rndHash,
		timestamp,
		builder.heightSpanKavaToBnb,
		rndNum,
	)
}

func getCurrentTimestamp() int64 { return time.Now().Unix() }

func getDeterministicRandomNumber() []byte {
	bytes := make([]byte, 32)
	if _, err := deterministicRand.Read(bytes); err != nil {
		panic(err)
	}
	return bytes
}

func convertBnbToKavaCoins(coins types.Coins) sdk.Coins {
	sdkCoins := sdk.NewCoins()
	for _, c := range coins {
		newDenom, ok := denomMap[c.Denom]
		if !ok {
			panic(fmt.Sprintf("unrecognized coin denom '%s'", c.Denom))
		}
		sdkCoins = sdkCoins.Add(sdk.NewInt64Coin(newDenom, c.Amount))
	}
	return sdkCoins
}

func convertKavaToBnbCoins(coins sdk.Coins) types.Coins {
	bnbCoins := types.Coins{}
	for _, c := range coins {
		newDenom, ok := reverseStringMap(denomMap)[c.Denom]
		if !ok {
			panic(fmt.Sprintf("unrecognized coin denom '%s'", c.Denom))
		}
		if !c.Amount.IsInt64() {
			panic(fmt.Sprintf("coin amount '%s' cannot be converted to int64", c.Amount))
		}
		bnbCoins = bnbCoins.Plus(types.Coins{types.Coin{Denom: newDenom, Amount: c.Amount.Int64()}})
	}
	return bnbCoins.Sort()
}

func reverseStringMap(m map[string]string) map[string]string {
	reversedMap := make(map[string]string, len(m))
	for k, v := range m {
		reversedMap[v] = k
	}
	return reversedMap
}
