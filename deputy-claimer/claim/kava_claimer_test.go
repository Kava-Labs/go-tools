package claim

import (
	"encoding/hex"
	"testing"
	"time"

	bnbtypes "github.com/binance-chain/go-sdk/common/types"
	bnbmsg "github.com/binance-chain/go-sdk/types/msg"
	"github.com/golang/mock/gomock"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/claim/mock"
)

var (
	addressesKavaDeputy sdk.AccAddress
	addressesKavaUsers0 sdk.AccAddress
	addressesBnbDeputy  bnbtypes.AccAddress
	addressesBnbUsers0  bnbtypes.AccAddress

	timestamp   int64
	rndNum0     []byte
	rndNum1     []byte
	rndNumHash0 []byte
	rndNumHash1 []byte

	testKavaSwaps bep3.AtomicSwaps
	testBnbSwaps  []bnbtypes.AtomicSwap
)

func init() {

	cfg := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(cfg)
	cfg.Seal()

	// These addresses are copied from kvtool common addresses.
	addressesKavaDeputy = mustDecodeKavaAddress("kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm")
	addressesKavaUsers0 = mustDecodeKavaAddress("kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc")
	addressesBnbDeputy = mustDecodeBnbAddress("bnb1zfa5vmsme2v3ttvqecfleeh2xtz5zghh49hfqe")
	addressesBnbUsers0 = mustDecodeBnbAddress("bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x")

	timestamp = time.Date(2020, 10, 11, 17, 0, 0, 0, time.UTC).Add(-10 * time.Minute).Unix() // TODO replace with fixed time once time is abstracted from claimer
	rndNum0 = mustDecodeHex("52af03e28b32dc838c98936a7654996bd21bcc0d3da5277d5065cf242b26dfe5")
	rndNum1 = mustDecodeHex("ed9895055b27771b8584de0e838a33d21b6b735de7fd6640770e877b1c23ae5f")

	rndNumHash0 = bep3.CalculateRandomHash(rndNum0, timestamp)
	rndNumHash1 = bep3.CalculateRandomHash(rndNum1, timestamp)

	testKavaSwaps = bep3.AtomicSwaps{
		{
			// swap ID ac36859ba07ec81123f7d860ce2ca6a704385bd3ace6654601d43f84a235d306
			Amount:              sdk.NewCoins(sdk.NewInt64Coin("bnb", 1_00_000_000)),
			RandomNumberHash:    rndNumHash0,
			ExpireHeight:        1_000_000,
			Timestamp:           timestamp,
			Sender:              addressesKavaUsers0,
			Recipient:           addressesKavaDeputy,
			SenderOtherChain:    addressesBnbDeputy.String(),
			RecipientOtherChain: addressesBnbUsers0.String(),
			ClosedBlock:         0, // default for open swaps
			Status:              bep3.Open,
			CrossChain:          true,
			Direction:           bep3.Outgoing,
		},
		// { // TODO remove
		// 	Amount:              sdk.NewCoins(sdk.NewInt64Coin("bnb", 1_00_000_000)),
		// 	RandomNumberHash:    rndNumHash1,
		// 	ExpireHeight:        1_000_000,
		// 	Timestamp:           timestamp,
		// 	Sender:              addressesKavaDeputy,
		// 	Recipient:           addressesKavaUsers0,
		// 	SenderOtherChain:    addressesBnbUsers0.String(),
		// 	RecipientOtherChain: addressesBnbDeputy.String(),
		// 	ClosedBlock:         0, // default for open swaps
		// 	Status:              bep3.Open,
		// 	CrossChain:          true,
		// 	Direction:           bep3.Incoming,
		// },
	}
	testBnbSwaps = []bnbtypes.AtomicSwap{
		{
			// kava to bnb swaps
			From:                addressesBnbDeputy,
			To:                  addressesBnbUsers0,
			OutAmount:           bnbtypes.Coins{{Denom: "BNB", Amount: 1_00_000_000}},
			InAmount:            nil, // seems to always be nil
			ExpectedIncome:      "100000000:BNB",
			RecipientOtherChain: addressesKavaDeputy.String(),
			RandomNumberHash:    rndNumHash0,
			RandomNumber:        rndNum0,
			Timestamp:           timestamp,
			CrossChain:          true,
			ExpireHeight:        10_000_000, // TODO
			Index:               0,          // TODO what is this?
			ClosedTime:          9_999_000,
			Status:              bnbtypes.Completed,
		},
		// { // TODO remove
		// 	// bnb to kava swap
		// 	From:                addressesBnbUsers0,
		// 	To:                  addressesBnbDeputy,
		// 	OutAmount:           bnbtypes.Coins{{Denom: "BNB", Amount: 1_00_000_000}},
		// 	InAmount:            nil, // seems to always be nil
		// 	ExpectedIncome:      "100000000:BNB",
		// 	RecipientOtherChain: addressesKavaUsers0.String(),
		// 	RandomNumberHash:    rndNumHash1,
		// 	RandomNumber:        nil, // default for unclaimed swap
		// 	Timestamp:           timestamp,
		// 	CrossChain:          true,
		// 	ExpireHeight:        10_000_000, // TODO
		// 	Index:               1,          // TODO what is this?
		// 	ClosedTime:          0,          // TODO default for open swaps?
		// 	Status:              bnbtypes.Open,
		// },
	}
}
func TestGetClaimableKavaSwaps(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kc := mock.NewMockKavaChainClient(ctrl)
	bc := mock.NewMockBnbChainClient(ctrl)

	kc.EXPECT().
		GetOpenOutgoingSwaps().
		Return(testKavaSwaps, nil)

	bc.EXPECT().
		GetRandomNumberFromSwap([]byte(calcBnbSwapID(testBnbSwaps[0], addressesKavaUsers0.String()))).
		Return(testBnbSwaps[0].RandomNumber, nil)

	swaps, err := getClaimableKavaSwaps(kc, bc, addressesBnbDeputy)
	require.NoError(t, err)

	expectedClaimableSwaps := []claimableSwap{
		{
			swapID:       testKavaSwaps[0].GetSwapID(),
			randomNumber: []byte(testBnbSwaps[0].RandomNumber),
		},
	}
	require.Equal(t, expectedClaimableSwaps, swaps)
}

func calcBnbSwapID(swap bnbtypes.AtomicSwap, senderOtherChain string) bnbtypes.SwapBytes {
	return bnbmsg.CalculateSwapID(swap.RandomNumberHash, swap.From, senderOtherChain)
}

func mustDecodeKavaAddress(address string) sdk.AccAddress {
	aa, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return aa
}

func mustDecodeBnbAddress(address string) bnbtypes.AccAddress {
	aa, err := bnbtypes.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return aa
}

func mustDecodeHex(hexString string) []byte {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		panic(err)
	}
	return bz
}
