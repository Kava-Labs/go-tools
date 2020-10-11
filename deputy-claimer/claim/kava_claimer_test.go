package claim

// import (
// 	"encoding/hex"
// 	"testing"

// 	"github.com/golang/mock"
// 	"github.com/golang/mock/gomock"
// 	"github.com/kava-labs/go-sdk/kava/bep3"
// 	"github.com/stretchr/testify/require"
// 	sdk "github.com/kava-labs/cosmos-sdk/types"

// 	"github.com/kava-labs/go-tools/deputy-claimer/claim/mock"
// )

// Generate testing mocks
//go:generate mockgen -source kava_client.go -destination mock/kava_client.go
//go:generate mockgen -source bnb_client.go -destination mock/bnb_client.go

// func TestFetchAndClaimSwaps(t *testing.T) {
// 	/*
// 		too big
// 	*/
// }
// func TestGetClaimableKavaSwaps(t *testing.T) {

// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	kc := mock.NewMockkavaChainClient(ctrl)
// 	bc := mock.NewMockbnbChainClient(ctrl)

// 	kc.EXPECT().
// 		getOpenSwaps().
// 		Return(getExampleSwaps())

// 	bc.EXPECT().
// 		getSwapByID(gomock.Eq()).
// 		Return()

// 	swaps, err := getClaimableKavaSwaps(kc, bc, nil)
// 	require.NoError(t, err)
// 	t.Log(swaps)
// }

// func getExampleSwaps() bep3.AtomicSwaps {
// 	rndNumHash, err := hex.DecodeString("464105c245199d02a4289475b8b231f3f73918b6f0fdad898825186950d46f36")
// 	if err != nil {
// 		panic(err)
// 	}

// 	return bep3.AtomicSwaps{
// 		{Amount: sdk.NewCoins("bnb", 1_00_000_000)              ,
// 			RandomNumberHash    : rndNumHash,
// 			ExpireHeight: 1_000_000        ,
// 			Timestamp: 1602285615,
// 			Sender:
// 			Recipient
// 			SenderOtherChain
// 			RecipientOtherChain
// 			ClosedBlock
// 			Status
// 			CrossChain
// 			Direction           },
// 	}
// }

// func mustGetAddressFromBech32(address string) sdk.AccAddress {
// 	a, err := sdk.GetAccAddressFromBech32(address)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return a
// }
