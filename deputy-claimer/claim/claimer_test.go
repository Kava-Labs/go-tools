package claim

import (
	"encoding/hex"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bnbmsg "github.com/kava-labs/binance-chain-go-sdk/types/msg"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/kava-labs/go-tools/deputy-claimer/claim/mock"
)

var (
	addressesKavaDeputy sdk.AccAddress
	addressesKavaUsers0 sdk.AccAddress
	addressesBnbDeputy  bnbtypes.AccAddress
	addressesBnbUsers0  bnbtypes.AccAddress

	mnemonicsKavaUsers0 = "season bone lucky dog depth pond royal decide unknown device fruit inch clock trap relief horse morning taxi bird session throw skull avocado private"

	timestamp   int64
	rndNum0     []byte
	rndNum1     []byte
	rndNumHash0 []byte
	rndNumHash1 []byte

	testKavaSwaps bep3types.AtomicSwaps
	testBnbSwaps  []bnbtypes.AtomicSwap
)

func init() {

	cfg := sdk.GetConfig()
	app.SetBech32AddressPrefixes(cfg)
	cfg.Seal()

	// These addresses are copied from kvtool common addresses.
	addressesKavaDeputy = mustDecodeKavaAddress("kava1agcvt07tcw0tglu0hmwdecsnuxp2yd45f3avgm")
	addressesKavaUsers0 = mustDecodeKavaAddress("kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc")
	addressesBnbDeputy = mustDecodeBnbAddress("bnb1zfa5vmsme2v3ttvqecfleeh2xtz5zghh49hfqe")
	addressesBnbUsers0 = mustDecodeBnbAddress("bnb10rr5f8m73rxgnz9afvnfn7fn9pwhfskem5kn0x")

	timestamp = time.Date(2020, 10, 11, 17, 0, 0, 0, time.UTC).Add(-10 * time.Minute).Unix() // TODO replace with fixed time once time is abstracted from claimer
	rndNum0 = mustDecodeHex("52af03e28b32dc838c98936a7654996bd21bcc0d3da5277d5065cf242b26dfe5")
	rndNum1 = mustDecodeHex("ed9895055b27771b8584de0e838a33d21b6b735de7fd6640770e877b1c23ae5f")

	rndNumHash0 = bep3types.CalculateRandomHash(rndNum0, timestamp)
	rndNumHash1 = bep3types.CalculateRandomHash(rndNum1, timestamp)

	testKavaSwaps = bep3types.AtomicSwaps{
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
			Status:              bep3types.Open,
			CrossChain:          true,
			Direction:           bep3types.Outgoing,
		},
		{
			Amount:              sdk.NewCoins(sdk.NewInt64Coin("bnb", 1_00_000_000)),
			RandomNumberHash:    rndNumHash1,
			ExpireHeight:        1_000_000,
			Timestamp:           timestamp,
			Sender:              addressesKavaDeputy,
			Recipient:           addressesKavaUsers0,
			SenderOtherChain:    addressesBnbUsers0.String(),
			RecipientOtherChain: addressesBnbDeputy.String(),
			ClosedBlock:         0, // default for open swaps
			Status:              bep3types.Open,
			CrossChain:          true,
			Direction:           bep3types.Incoming,
		},
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
		{
			// bnb to kava swap
			From:                addressesBnbUsers0,
			To:                  addressesBnbDeputy,
			OutAmount:           bnbtypes.Coins{{Denom: "BNB", Amount: 1_00_000_000}},
			InAmount:            nil, // seems to always be nil
			ExpectedIncome:      "100000000:BNB",
			RecipientOtherChain: addressesKavaUsers0.String(),
			RandomNumberHash:    rndNumHash1,
			RandomNumber:        nil, // default for unclaimed swap
			Timestamp:           timestamp,
			CrossChain:          true,
			ExpireHeight:        10_000_000, // TODO
			Index:               1,          // TODO what is this?
			ClosedTime:          0,          // TODO default for open swaps?
			Status:              bnbtypes.Open,
		},
	}
}
func TestGetClaimableKavaSwaps(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	kc := mock.NewMockKavaChainClient(ctrl)
	bc := mock.NewMockBnbChainClient(ctrl)

	kc.EXPECT().
		GetOpenOutgoingSwaps().
		Return(testKavaSwaps[:1], nil) // only return outgoing swaps

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

func TestGetClaimableBnbSwaps(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bc := mock.NewMockBnbChainClient(ctrl)
	kc := mock.NewMockKavaChainClient(ctrl)

	bc.EXPECT().
		GetOpenOutgoingSwaps().
		Return(testBnbSwaps[1:], nil) // return only outgoing swaps

	kc.EXPECT().
		GetRandomNumberFromSwap([]byte(testKavaSwaps[1].GetSwapID())).
		Return(rndNum1, nil)

	swaps, err := getClaimableBnbSwaps(kc, bc, addressesKavaDeputy)
	require.NoError(t, err)

	expectedClaimableSwaps := []claimableSwap{
		{
			swapID:       []byte(calcBnbSwapID(testBnbSwaps[1], addressesKavaDeputy.String())),
			randomNumber: rndNum1,
		},
	}
	require.Equal(t, expectedClaimableSwaps, swaps)
}

func TestConstructAndSendKavaClaim(t *testing.T) {
	// setup mock client
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	kc := mock.NewMockKavaChainClient(ctrl)

	// set endpoints to return testing data
	cdc := app.MakeCodec()
	kc.EXPECT().
		GetCodec().
		Return(cdc).AnyTimes()
	kc.EXPECT().
		GetChainID().
		Return("kava-localnet", nil).AnyTimes()
	testAcc := authexported.Account(&authtypes.BaseAccount{
		Address:       addressesKavaUsers0,
		AccountNumber: 12,
		Sequence:      34,
	})
	kc.EXPECT().
		GetAccount(addressesKavaUsers0).
		Return(testAcc, nil).AnyTimes()

	expectedTxJSON := `{
		"type": "cosmos-sdk/StdTx",
		"value": {
			"msg": [
				{
					"type": "bep3/MsgClaimAtomicSwap",
					"value": {
						"from": "kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc",
						"swap_id": "9E3FDDA337B885622E8C0C6A7970C95BC312A97BB7BA38C26F0E3D7A44FB93A8",
						"random_number": "6712DDF02589858704CF70CF39FFF8724FE71F1F2D7560878A97BBC5C1367535"
					}
				}
			],
			"fee": {
				"amount": [],
				"gas": "250000"
			},
			"signatures": [
				{
					"pub_key": {
						"type": "tendermint/PubKeySecp256k1",
						"value": "AuHcgEkmL+Ed4ZjXPDSLRQxmNotxh/l8hBJCi2EvZIh1"
					},
					"signature": "0w+31XqrpS8ZpG0piYLL+ItQMIEbqJnkOUJnQZaAKQoAHzrJKWqBex9xd+I9yw82/SN0sCpffJwViTx8K5FPVA=="
				}
			],
			"memo": ""
		}
	}`
	var expectedTx authtypes.StdTx
	cdc.MustUnmarshalJSON([]byte(expectedTxJSON), &expectedTx)
	expectedTxBytes := tmtypes.Tx(cdc.MustMarshalBinaryLengthPrefixed(expectedTx))
	// set expected tx
	kc.EXPECT().BroadcastTx(expectedTxBytes)

	// run function under test (mock will verify tx was created correctly)
	testID := mustDecodeHex("9e3fdda337b885622e8c0c6a7970c95bc312a97bb7ba38c26f0e3d7a44fb93a8")
	testRndNum := mustDecodeHex("6712ddf02589858704cf70cf39fff8724fe71f1f2d7560878a97bbc5c1367535")
	_, err := constructAndSendClaim(kc, mnemonicsKavaUsers0, testID, testRndNum)
	require.NoError(t, err)
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
