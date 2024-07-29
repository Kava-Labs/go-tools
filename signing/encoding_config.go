package signing

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/kava-labs/kava/app/params"
)

type EncodingConfigAdapter struct {
	params.EncodingConfig
}

func (e EncodingConfigAdapter) InterfaceRegistry() types.InterfaceRegistry {
	return e.EncodingConfig.InterfaceRegistry
}

func (e EncodingConfigAdapter) Marshaler() codec.Codec {
	return e.EncodingConfig.Marshaler
}

func (e EncodingConfigAdapter) TxConfig() client.TxConfig {
	return e.EncodingConfig.TxConfig
}

func (e EncodingConfigAdapter) Amino() *codec.LegacyAmino {
	return e.EncodingConfig.Amino
}
