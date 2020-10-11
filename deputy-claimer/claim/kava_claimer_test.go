package claim

import (
	"encoding/hex"
	"os"
	"testing"

	bnbtypes "github.com/binance-chain/go-sdk/common/types"
	"github.com/golang/mock/gomock"
	sdk "github.com/kava-labs/cosmos-sdk/types"
	"github.com/kava-labs/go-sdk/kava"
	"github.com/kava-labs/go-sdk/kava/bep3"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/claim/mock"
)

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(cfg)
	cfg.Seal()

	os.Exit(m.Run())
}

func TestGetClaimableKavaSwaps(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kc := mock.NewMockKavaChainClient(ctrl)
	bc := mock.NewMockBnbChainClient(ctrl)

	kc.EXPECT().
		GetOpenSwaps().
		Return(getExampleSwapsKava(), nil)

	bc.EXPECT().
		GetSwapByID(gomock.Any()). // TODO filter for: mustDecodeHex("bc5608faba3c12a852fd74fe21bfeb12d86f1d5a8a0b2919f221074454801368")).
		Return(getExampleSwapsBnb()[0], nil)
	bc.EXPECT().
		GetSwapByID(gomock.Any()).         // TODO filter for other the other swap ID
		Return(bnbtypes.AtomicSwap{}, nil) // TODO fill out real swap with no random number

	swaps, err := getClaimableKavaSwaps(kc, bc, mustDecodeBnbAddress("bnb1z8ryd66lhc4d9c0mmxx9zyyq4t3cqht9mt0qz3")) // bnb deputy hot wallet
	require.NoError(t, err)

	expectedClaimableSwaps := []claimableSwap{
		{
			swapID:       mustDecodeHex("ac36859ba07ec81123f7d860ce2ca6a704385bd3ace6654601d43f84a235d306"),
			randomNumber: []byte(getExampleSwapsBnb()[0].RandomNumber),
		},
	}
	require.Equal(t, expectedClaimableSwaps, swaps)
}

func getExampleSwapsKava() bep3.AtomicSwaps {

	return bep3.AtomicSwaps{
		{
			// swap ID ac36859ba07ec81123f7d860ce2ca6a704385bd3ace6654601d43f84a235d306
			Amount:              sdk.NewCoins(sdk.NewInt64Coin("bnb", 1_00_000_000)),
			RandomNumberHash:    mustDecodeHex("3a747b6a684b708ffb01e483d5ae765ac927763f576e22eb3b201833c6f06f5a"),
			ExpireHeight:        1_000_000,
			Timestamp:           1602285615,
			Sender:              mustDecodeKavaAddress("kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc"), // user0
			Recipient:           mustDecodeKavaAddress("kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm"), // bnb deputy hot wallet
			SenderOtherChain:    "bnb1z8ryd66lhc4d9c0mmxx9zyyq4t3cqht9mt0qz3",                         // bnb deputy hot wallet
			RecipientOtherChain: "bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x",                         // user0
			ClosedBlock:         0,                                                                    // default for open swaps
			Status:              bep3.Open,
			CrossChain:          true,
			Direction:           bep3.Outgoing,
		},
		{
			Amount:              sdk.NewCoins(sdk.NewInt64Coin("bnb", 1_00_000_000)),
			RandomNumberHash:    mustDecodeHex("7e4b2a830a568480f1568d28059e3111c179f86726b1ba7362177a950210d443"),
			ExpireHeight:        1_000_100,
			Timestamp:           1602286615,
			Sender:              mustDecodeKavaAddress("kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm"), // bnb deputy hot wallet
			Recipient:           mustDecodeKavaAddress("kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc"), // user0
			SenderOtherChain:    "bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x",                         // user0
			RecipientOtherChain: "bnb1z8ryd66lhc4d9c0mmxx9zyyq4t3cqht9mt0qz3",                         // bnb deputy hot wallet
			ClosedBlock:         0,                                                                    // default for open swaps
			Status:              bep3.Open,
			CrossChain:          true,
			Direction:           bep3.Incoming,
		},
	}
}

func getExampleSwapsBnb() []bnbtypes.AtomicSwap {
	return []bnbtypes.AtomicSwap{
		{
			// swap id bc5608faba3c12a852fd74fe21bfeb12d86f1d5a8a0b2919f221074454801368
			From: mustDecodeBnbAddress("bnb1z8ryd66lhc4d9c0mmxx9zyyq4t3cqht9mt0qz3"), // bnb deputy hot wallet
			To:   mustDecodeBnbAddress("bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x"), // user0
			// OutAmount: Coins        // TODO
			// InAmount:  Coins        // TODO
			// ExpectedIncome:  string // TODO
			RecipientOtherChain: "kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm", // bnb deputy hot wallet
			RandomNumberHash:    mustDecodeHex("3a747b6a684b708ffb01e483d5ae765ac927763f576e22eb3b201833c6f06f5a"),
			RandomNumber:        mustDecodeHex("52af03e28b32dc838c98936a7654996bd21bcc0d3da5277d5065cf242b26dfe5"),
			Timestamp:           1602285615,
			CrossChain:          true,
			ExpireHeight:        10_000_000,
			Index:               0, // TODO what is this?
			ClosedTime:          9_999_000,
			Status:              bnbtypes.Completed,
		},
		{
			// nil swap which will have random number 0, // TODO fill this out
		},
	}
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
