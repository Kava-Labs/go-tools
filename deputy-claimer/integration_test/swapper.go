package integrationtest

import (
	"fmt"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbKeys "github.com/kava-labs/binance-chain-go-sdk/keys"
	"github.com/kava-labs/binance-chain-go-sdk/types/msg"
	"github.com/kava-labs/go-sdk/client"
	"github.com/kava-labs/go-sdk/keys"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"

	"github.com/kava-labs/go-tools/deputy-claimer/integration_test/common"
)

// type CrossChainSwap struct {
// 	Sender, Receiver fmt.Stringer
// 	Amount           int64
// 	Denom            string // reference lookup table

// 	KavaDeputy, BnbDeputy fmt.Stringer

// 	randomNumber []byte
// 	timestamp    int64
// }

// func (CrossChainSwap) SendInitialSwap(commit string) {}

// type SwapCreator struct {
// 	BnbDeputy  types.AccAddress
// 	KavaDeputy sdk.AccAddress

// 	bnbClient  *bnbRpc.HTTP
// 	kavaClient *client.KavaClient
// }

// func (sc SwapCreator) SubmitBnbSwap(sender types.AccAddress, receiver sdk.AccAddress, rndHash []byte, timestamp int64, coins types.Coins, expectedIncome string, heightSpan int64, broadcastMode bnbRpc.SyncType) {
// 	sc.bnbClient.HTLT(
// 		addrs.Bnb.Deputys.Bnb.HotWallet.Address,           // recipient
// 		addrs.Kava.Users[0].Address.String(),              // recipient other chain
// 		addrs.Kava.Deputys.Bnb.HotWallet.Address.String(), // other chain sender
// 		rndHash,
// 		timestamp,
// 		types.Coins{{Denom: "BNB", Amount: 500_000_000}}, //{Denom: "BNB", Amount: 100_000_000}},
// 		"",  // expected income
// 		360, // heightspan
// 		true,
// 		bnbRpc.Commit,
// 	)
// }
// func (sc SwapCreator) SubmitKavaSwap(sender sdk.AccAddress, receiver types.AccAddress, rndHash []byte, timestamp int64, coins sdk.Coins, heightSpan int64, broadcastMode client.SyncType) {
// }

/*
how I think about swaps (in the context of these tests)

send swap, on kava (chain), from user (direction)
sendOutgoingKavaSwap(broadcastMode) // options: timestamp, rndNum, coins, heightspan

setup swap from kava to bnb (send swaps on both chains)

want a default, but add changes from default - functional options, many functions
already have the base layer - sendKava/BnbSwap(........)
swapCreator.sendOutgoingKavaSwap(.......) fixes depRec, depSen
defaultSwapCreator.sendOutgoingKavaSwap(opts...)
*/

type KavaSwap struct {
	bep3types.AtomicSwap
	SenderMnemonic string
	HeightSpan     uint64
}

func NewKavaSwap(senderMnemonic string, recipient sdk.AccAddress, senderOtherChain, recipientOtherChain string, amount sdk.Coins, timestamp int64, rndHash []byte, heightspan int64) KavaSwap {
	if heightspan < 0 {
		panic("heightspan cannot be negative")
	}
	return KavaSwap{
		AtomicSwap: bep3types.AtomicSwap{
			Amount:              amount,
			RandomNumberHash:    rndHash,
			Timestamp:           timestamp,
			Sender:              kavaAddressFromMnemonic(senderMnemonic),
			Recipient:           recipient, // TODO name?
			SenderOtherChain:    senderOtherChain,
			RecipientOtherChain: recipientOtherChain,
		},
		SenderMnemonic: senderMnemonic,
		HeightSpan:     uint64(heightspan),
	}
}
func (swap KavaSwap) Broadcast(mode client.SyncType) ([]byte, error) {
	cdc := app.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, swap.SenderMnemonic, app.Bip44CoinType, common.KavaNodeURL) // TODO add to swap struct?
	createMsg := bep3types.NewMsgCreateAtomicSwap(
		swap.Sender,
		swap.Recipient,
		swap.RecipientOtherChain,
		swap.SenderOtherChain,
		swap.RandomNumberHash,
		swap.Timestamp,
		swap.Amount,
		swap.HeightSpan,
	)
	res, err := kavaClient.Broadcast(createMsg, mode)
	if err != nil {
		return res.Hash, err
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}
func (swap KavaSwap) SubmitClaim(randomNumber []byte, mode client.SyncType) ([]byte, error) {
	cdc := app.MakeCodec()
	kavaClient := client.NewKavaClient(cdc, swap.SenderMnemonic, app.Bip44CoinType, common.KavaNodeURL) // TODO add to swap struct?

	msg := bep3types.NewMsgClaimAtomicSwap(
		swap.Sender, // TODO doesn't need to be sender
		swap.GetSwapID(),
		randomNumber,
	)
	res, err := kavaClient.Broadcast(msg, client.Commit)
	if err != nil {
		return res.Hash, err
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf("tx rejected from chain: %s", res.Log)
	}
	return res.Hash, nil
}

// refund
// claim(rndNum)

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
func (swap BnbSwap) Broadcast(mode bnbRpc.SyncType) ([]byte, error) {
	bnbClient := NewBnbClient(swap.SenderMnemonic, common.BnbNodeURL) // TODO
	res, err := bnbClient.HTLT(
		swap.To,
		swap.RecipientOtherChain,
		swap.SenderOtherChain,
		swap.RandomNumberHash,
		swap.Timestamp,
		swap.InAmount,
		swap.ExpectedIncome,
		swap.HeightSpan,
		swap.CrossChain,
		mode,
	)
	if err != nil {
		return res.Hash, err
	}
	if res.Code != 0 {
		return res.Hash, fmt.Errorf(res.Log) // TODO
	}
	return res.Hash, nil
}
func (swap BnbSwap) GetSwapID() []byte {
	return msg.CalculateSwapID(swap.RandomNumberHash, swap.From, swap.SenderOtherChain)
}

// type SwapCreator struct {
// 	KavaDeputyMnemonic string
// 	BnbDeputyMnemonic string
// }
// func (SwapCreator) NewOutgoingKavaSwap(.....) KavaSwap {
// 	// NewKavaSwap(... )
// 	// Recipient: dep
// 	// SenderOtherChain: dep
// }
// func (SwapCreator) NewIncomingKavaSwap() KavaSwap {
// 	//
// }

// func (SwapCreator) NewDefaultOutgoingKavaSwap(opts) KavaSwap {}
// or do swap.WithHeightSpan(234)

// setup kava2bnbSwap

func kavaAddressFromMnemonic(mnemonic string) sdk.AccAddress {
	keyManager, err := keys.NewMnemonicKeyManager(mnemonic, app.Bip44CoinType)
	if err != nil {
		panic(fmt.Sprintf("new key manager from mnenomic err, err=%s", err.Error())) // TODO
	}
	return keyManager.GetAddr()
}
func bnbAddressFromMnemonic(mnemonic string) types.AccAddress {
	keyManager, err := bnbKeys.NewMnemonicKeyManager(mnemonic)
	if err != nil {
		panic(err)
	}
	return keyManager.GetAddr()
}
