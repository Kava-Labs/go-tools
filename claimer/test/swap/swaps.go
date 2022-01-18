package swap

import (
	"fmt"

	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/binance-chain-go-sdk/types/msg"
	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
)

// KavaSwap holds all parameters required to create a HTLT on the kava chain.
type KavaSwap struct {
	bep3types.AtomicSwap
	SenderMnemonic string
	HeightSpan     uint64
}

func NewKavaSwap(senderMnemonic string, recipient sdk.AccAddress, senderOtherChain, recipientOtherChain string, amount sdk.Coins, timestamp int64, rndHash []byte, heightspan int64) KavaSwap {
	return KavaSwap{
		AtomicSwap: bep3types.AtomicSwap{
			Amount:              amount,
			RandomNumberHash:    rndHash,
			Timestamp:           timestamp,
			Sender:              kavaAddressFromMnemonic(senderMnemonic),
			Recipient:           recipient,
			SenderOtherChain:    senderOtherChain,
			RecipientOtherChain: recipientOtherChain,
		},
		SenderMnemonic: senderMnemonic,
		HeightSpan:     uint64(heightspan),
	}
}

// BnbSwap holds all parameters required to create a HTLT on the bnb chain.
type BnbSwap struct {
	types.AtomicSwap
	SenderMnemonic   string
	SenderOtherChain string
	HeightSpan       int64
}

func NewBnbSwap(senderMnemonic string, recipient types.AccAddress, senderOtherChain, recipientOtherChain string, amount types.Coins, timestamp int64, rndHash []byte, heightSpan int64) BnbSwap {
	return BnbSwap{
		AtomicSwap: types.AtomicSwap{
			From:                bnbAddressFromMnemonic(senderMnemonic),
			To:                  recipient,
			RecipientOtherChain: recipientOtherChain,
			InAmount:            amount,
			RandomNumberHash:    rndHash,
			Timestamp:           timestamp,
			CrossChain:          true,
		},
		SenderMnemonic:   senderMnemonic,
		SenderOtherChain: senderOtherChain,
		HeightSpan:       heightSpan,
	}
}

func (swap BnbSwap) GetSwapID() []byte {
	return msg.CalculateSwapID(swap.RandomNumberHash, swap.From, swap.SenderOtherChain)
}

// CrossChainSwap holds details of both swaps involved in moving assets from one chain to the other.
type CrossChainSwap struct {
	KavaSwap     KavaSwap
	BnbSwap      BnbSwap
	RandomNumber []byte
}

// NewBnbToKavaSwap creates valid bnb and kava swaps to move assets from bnb to kava chains.
func NewBnbToKavaSwap(senderMnemonic string, recipient sdk.AccAddress, amount SwapAmount, kavaDeputyMnemonic string, bnbDeputyAddress types.AccAddress, rndHash []byte, timestamp int64, heightSpan SwapHeightSpan, rndNum []byte) CrossChainSwap {
	return CrossChainSwap{
		BnbSwap: NewBnbSwap(
			senderMnemonic,
			bnbDeputyAddress,
			kavaAddressFromMnemonic(kavaDeputyMnemonic).String(),
			recipient.String(),
			amount.Bnb,
			timestamp,
			rndHash,
			heightSpan.Bnb,
		),
		KavaSwap: NewKavaSwap(
			kavaDeputyMnemonic,
			recipient,
			bnbAddressFromMnemonic(senderMnemonic).String(),
			bnbDeputyAddress.String(),
			amount.Kava,
			timestamp,
			rndHash,
			heightSpan.Kava,
		),
		RandomNumber: rndNum,
	}
}

// NewKavaToBnbSwap creates valid kava and bnb swaps to move assets from kava to bnb chains.
func NewKavaToBnbSwap(senderMnemonic string, recipient types.AccAddress, amount SwapAmount, bnbDeputyMnemonic string, kavaDeputyAddress sdk.AccAddress, rndHash []byte, timestamp int64, heightSpan SwapHeightSpan, rndNum []byte) CrossChainSwap {
	return CrossChainSwap{
		KavaSwap: NewKavaSwap(
			senderMnemonic,
			kavaDeputyAddress,
			bnbAddressFromMnemonic(bnbDeputyMnemonic).String(),
			recipient.String(),
			amount.Kava,
			timestamp,
			rndHash,
			heightSpan.Kava,
		),
		BnbSwap: NewBnbSwap(
			bnbDeputyMnemonic,
			recipient,
			kavaAddressFromMnemonic(senderMnemonic).String(),
			kavaDeputyAddress.String(),
			amount.Bnb,
			timestamp,
			rndHash,
			heightSpan.Bnb,
		),
		RandomNumber: rndNum,
	}
}

type SwapAmount struct {
	Kava sdk.Coins
	Bnb  types.Coins
}

type SwapHeightSpan struct {
	Kava, Bnb int64
}

func kavaAddressFromMnemonic(mnemonic string) sdk.AccAddress {
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
	if err != nil {
		panic(fmt.Sprintf("invalid mnemonic: %v", err.Error()))
	}
	return keyManager.GetKeyRing().GetAddress()
}

func bnbAddressFromMnemonic(mnemonic string) types.AccAddress {
	keyManager, err := bnbKeys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(fmt.Sprintf("invalid mnemonic: %v", err.Error()))
	}
	return keyManager.GetAddr()
}
