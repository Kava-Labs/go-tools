package claim

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
)

type DeputyAddress struct {
	Kava sdk.AccAddress
	Bnb  types.AccAddress
}

type DeputyAddresses map[string]DeputyAddress

func (deputyAddresses DeputyAddresses) AllKava() []sdk.AccAddress {
	var addresses []sdk.AccAddress
	for _, da := range deputyAddresses {
		addresses = append(addresses, da.Kava)
	}
	return addresses
}

func (deputyAddresses DeputyAddresses) AllBnb() []types.AccAddress {
	var addresses []types.AccAddress
	for _, da := range deputyAddresses {
		addresses = append(addresses, da.Bnb)
	}
	return addresses
}

func (deputyAddresses DeputyAddresses) GetMatchingBnb(kavaAddress sdk.AccAddress) (types.AccAddress, bool) {
	for _, da := range deputyAddresses {
		if da.Kava.Equals(kavaAddress) {
			return da.Bnb, true
		}
	}
	return nil, false
}

func (deputyAddresses DeputyAddresses) GetMatchingKava(bnbAddress types.AccAddress) (sdk.AccAddress, bool) {
	for _, da := range deputyAddresses {
		if bytes.Equal(da.Bnb.Bytes(), bnbAddress.Bytes()) {
			return da.Kava, true
		}
	}
	return nil, false
}
