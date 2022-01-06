package claim

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"
	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	bnbmsg "github.com/kava-labs/binance-chain-go-sdk/types/msg"
	"github.com/kava-labs/kava/app"
	bep3types "github.com/kava-labs/kava/x/bep3/types"
	tmtypes "github.com/kava-labs/tendermint/types"
	"github.com/stretchr/testify/require"

	"github.com/kava-labs/go-tools/deputy-claimer/claim/mock"
	"github.com/kava-labs/go-tools/deputy-claimer/test/addresses"
)

var (
	addrs    addresses.Addresses
	depAddrs DeputyAddresses

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

	addrs = addresses.GetAddresses()
	depAddrs = getDeputyAddresses(addrs)

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
			Sender:              addrs.Kava.Users[0].Address,
			Recipient:           addrs.Kava.Deputys.Bnb.HotWallet.Address,
			SenderOtherChain:    addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(),
			RecipientOtherChain: addrs.Bnb.Users[0].Address.String(),
			ClosedBlock:         0, // default for open swaps
			Status:              bep3types.SWAP_STATUS_OPEN,
			CrossChain:          true,
			Direction:           bep3types.SWAP_DIRECTION_OUTGOING,
		},
		{
			Amount:              sdk.NewCoins(sdk.NewInt64Coin("bnb", 1_00_000_000)),
			RandomNumberHash:    rndNumHash1,
			ExpireHeight:        1_000_000,
			Timestamp:           timestamp,
			Sender:              addrs.Kava.Deputys.Bnb.HotWallet.Address,
			Recipient:           addrs.Kava.Users[0].Address,
			SenderOtherChain:    addrs.Bnb.Users[0].Address.String(),
			RecipientOtherChain: addrs.Bnb.Deputys.Bnb.HotWallet.Address.String(),
			ClosedBlock:         0, // default for open swaps
			Status:              bep3types.SWAP_STATUS_OPEN,
			CrossChain:          true,
			Direction:           bep3types.SWAP_DIRECTION_OUTGOING,
		},
	}
	testBnbSwaps = []bnbtypes.AtomicSwap{
		{
			// kava to bnb swaps
			From:                addrs.Bnb.Deputys.Bnb.HotWallet.Address,
			To:                  addrs.Bnb.Users[0].Address,
			OutAmount:           bnbtypes.Coins{{Denom: "BNB", Amount: 1_00_000_000}},
			InAmount:            nil, // seems to always be nil
			ExpectedIncome:      "100000000:BNB",
			RecipientOtherChain: addrs.Kava.Deputys.Bnb.HotWallet.Address.String(),
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
			From:                addrs.Bnb.Users[0].Address,
			To:                  addrs.Bnb.Deputys.Bnb.HotWallet.Address,
			OutAmount:           bnbtypes.Coins{{Denom: "BNB", Amount: 1_00_000_000}},
			InAmount:            nil, // seems to always be nil
			ExpectedIncome:      "100000000:BNB",
			RecipientOtherChain: addrs.Kava.Users[0].Address.String(),
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
		Return(mapAtomicSwapsToResponses(testKavaSwaps[:1]), nil) // only return outgoing swaps

	bc.EXPECT().
		GetRandomNumberFromSwap([]byte(calcBnbSwapID(testBnbSwaps[0], addrs.Kava.Users[0].Address.String()))).
		Return(testBnbSwaps[0].RandomNumber, nil)

	swaps, err := getClaimableKavaSwaps(kc, bc, depAddrs)
	require.NoError(t, err)

	expectedClaimableSwaps := []kavaClaimableSwap{
		{
			swapID:       testKavaSwaps[0].GetSwapID(),
			destSwapID:   []byte(calcBnbSwapID(testBnbSwaps[0], addrs.Kava.Users[0].Address.String())),
			randomNumber: []byte(testBnbSwaps[0].RandomNumber),
			amount:       testKavaSwaps[0].Amount,
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

	swaps, err := getClaimableBnbSwaps(kc, bc, depAddrs)
	require.NoError(t, err)

	expectedClaimableSwaps := []bnbClaimableSwap{
		{
			swapID:       []byte(calcBnbSwapID(testBnbSwaps[1], addrs.Kava.Deputys.Bnb.HotWallet.Address.String())),
			destSwapID:   []byte(testKavaSwaps[1].GetSwapID()),
			randomNumber: rndNum1,
			amount:       testBnbSwaps[1].OutAmount,
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
	encodingConfig := app.MakeEncodingConfig()

	kc.EXPECT().
		GetChainID().
		Return("kava-localnet", nil).AnyTimes()

	testAcc := authtypes.AccountI(&authtypes.BaseAccount{
		Address:       addrs.Kava.Users[0].Address.String(),
		AccountNumber: 12,
		Sequence:      34,
	})
	kc.EXPECT().
		GetAccount(addrs.Kava.Users[0].Address).
		Return(testAcc, nil).AnyTimes()

	testID := mustDecodeBase64("nj/doze4hWIujAxqeXDJW8MSqXu3ujjCbw49ekT7k6g=")
	testRndNum := mustDecodeBase64("ZxLd8CWJhYcEz3DPOf/4ck/nHx8tdWCHipe7xcE2dTU=")

	expectedTxJSON := `
	{
		"body": {
			"messages": [
				{
					"@type": "/kava.bep3.v1beta1.MsgClaimAtomicSwap",
					"from": "kava173w2zz287s36ewnnkf4mjansnthnnsz7rtrxqc",
					"swap_id": "nj/doze4hWIujAxqeXDJW8MSqXu3ujjCbw49ekT7k6g=",
					"random_number": "ZxLd8CWJhYcEz3DPOf/4ck/nHx8tdWCHipe7xcE2dTU="
				}
			],
			"memo": "",
			"timeout_height": "0",
			"extension_options": [],
			"non_critical_extension_options": []
		},
		"auth_info": {
			"signer_infos": [
				{
					"public_key": {
					"@type": "/cosmos.crypto.secp256k1.PubKey",
					"key": "AuHcgEkmL+Ed4ZjXPDSLRQxmNotxh/l8hBJCi2EvZIh1"
				},
				"mode_info": {
					"single": {
						"mode": "SIGN_MODE_DIRECT"
					}
				},
				"sequence": "34"
				}
			],
			"fee": {
				"amount": [
					{
						"denom": "ukava",
						"amount": "62500"
					}
				],
				"gas_limit": "250000",
				"payer": "",
				"granter": ""
			}
		},
		"signatures": [
			"kzb57JAozylztqS+FHQ27JVLpoO++LNj8meaK5Gs0nBujrJJyhFILq8c0XdDQFTpV7pEeIt4EOIKt+Hezuxf6w=="
		]
	}
	`

	var expectedTx tx.Tx
	encodingConfig.Marshaler.MustUnmarshalJSON([]byte(expectedTxJSON), &expectedTx)
	expectedTxBytes := tmtypes.Tx(encodingConfig.Marshaler.MustMarshal(&expectedTx))

	broadcastTxRequest := tx.BroadcastTxRequest{
		TxBytes: expectedTxBytes,
		Mode:    tx.BroadcastMode_BROADCAST_MODE_SYNC,
	}
	// set expected tx
	kc.EXPECT().BroadcastTx(broadcastTxRequest)

	// run function under test (mock will verify tx was created correctly)
	_, err := constructAndSendClaim(kc, encodingConfig, mnemonicsKavaUsers0, testID, testRndNum)
	require.NoError(t, err)
}

func calcBnbSwapID(swap bnbtypes.AtomicSwap, senderOtherChain string) bnbtypes.SwapBytes {
	return bnbmsg.CalculateSwapID(swap.RandomNumberHash, swap.From, senderOtherChain)
}

func getDeputyAddresses(addrs addresses.Addresses) DeputyAddresses {
	return DeputyAddresses{
		"bnb": {
			Kava: addrs.Kava.Deputys.Bnb.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Bnb.HotWallet.Address,
		},
		"busd": {
			Kava: addrs.Kava.Deputys.Busd.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Busd.HotWallet.Address,
		},
		"btcb": {
			Kava: addrs.Kava.Deputys.Btcb.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Btcb.HotWallet.Address,
		},
		"xrpb": {
			Kava: addrs.Kava.Deputys.Xrpb.HotWallet.Address,
			Bnb:  addrs.Bnb.Deputys.Xrpb.HotWallet.Address,
		},
	}
}

func mustDecodeHex(hexString string) []byte {
	bz, err := hex.DecodeString(hexString)
	if err != nil {
		panic(err)
	}
	return bz
}

func mustDecodeBase64(hexString string) []byte {
	bz, err := base64.StdEncoding.DecodeString(hexString)
	if err != nil {
		panic(err)
	}
	return bz
}

func mapAtomicSwapsToResponses(atomicSwaps bep3types.AtomicSwaps) []bep3types.AtomicSwapResponse {
	var swapResponses []bep3types.AtomicSwapResponse

	for _, swap := range atomicSwaps {
		swapResponses = append(swapResponses, mapAtomicSwapToResponse(swap))
	}

	return swapResponses
}

func mapAtomicSwapToResponse(atomicSwap bep3types.AtomicSwap) bep3types.AtomicSwapResponse {
	return bep3types.AtomicSwapResponse{
		Id:                  atomicSwap.GetSwapID().String(),
		Amount:              atomicSwap.Amount,
		RandomNumberHash:    atomicSwap.RandomNumberHash.String(),
		ExpireHeight:        atomicSwap.ExpireHeight,
		Timestamp:           atomicSwap.Timestamp,
		Sender:              atomicSwap.Sender.String(),
		Recipient:           atomicSwap.Recipient.String(),
		SenderOtherChain:    atomicSwap.SenderOtherChain,
		RecipientOtherChain: atomicSwap.RecipientOtherChain,
		ClosedBlock:         atomicSwap.ClosedBlock,
		Status:              atomicSwap.Status,
		CrossChain:          atomicSwap.CrossChain,
		Direction:           atomicSwap.Direction,
	}
}
