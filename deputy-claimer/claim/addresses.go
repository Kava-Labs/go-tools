package claim

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
)

// DeputyAddress contains the kava and bnb addresses for a single deputy process.
type DeputyAddress struct {
	Kava sdk.AccAddress
	Bnb  types.AccAddress
}

// DeputyAddresses holds all on-chain addresses for multiple deputy processes.
// It provides convenience lookup methods to fetch addresses.
type DeputyAddresses map[string]DeputyAddress

// AllKava return the kava addresses for all deputies
func (deputyAddresses DeputyAddresses) AllKava() []sdk.AccAddress {
	var addresses []sdk.AccAddress
	for _, da := range deputyAddresses {
		addresses = append(addresses, da.Kava)
	}
	return addresses
}

// AllBnb return the bnb chain addresses for all deputies
func (deputyAddresses DeputyAddresses) AllBnb() []types.AccAddress {
	var addresses []types.AccAddress
	for _, da := range deputyAddresses {
		addresses = append(addresses, da.Bnb)
	}
	return addresses
}

// GetMatchingBnb finds the bnb chain address for the deputy with the provided kava address
func (deputyAddresses DeputyAddresses) GetMatchingBnb(kavaAddress sdk.AccAddress) (types.AccAddress, bool) {
	for _, da := range deputyAddresses {
		if da.Kava.Equals(kavaAddress) {
			return da.Bnb, true
		}
	}
	return nil, false
}

// GetMatchingKava finds the kava address for the deputy with the provided bnb chain address
func (deputyAddresses DeputyAddresses) GetMatchingKava(bnbAddress types.AccAddress) (sdk.AccAddress, bool) {
	for _, da := range deputyAddresses {
		if bytes.Equal(da.Bnb.Bytes(), bnbAddress.Bytes()) {
			return da.Kava, true
		}
	}
	return nil, false
}
