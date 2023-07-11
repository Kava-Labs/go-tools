package config

import (
	"github.com/kava-labs/go-tools/deputy-claimer/claim"
	"github.com/spf13/viper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
)

type Config struct {
	BnbRPCURL     string
	KavaGrpcURL   string
	Deputies      claim.DeputyAddresses
	BnbMnemonics  []string
	KavaMnemonics []string
}

type ConfigSimple struct {
	BnbRPCURL   string
	KavaGrpcURL string
	Deputies    map[string]struct {
		Kava string
		Bnb  string
	}
	BnbMnemonics  []string
	KavaMnemonics []string
}

func LoadConfig() (Config, error) {
	v := viper.New()
	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("toml")
	v.AddConfigPath("$HOME")
	if err := v.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var temp ConfigSimple
	if err := v.Unmarshal(&temp); err != nil {
		return Config{}, err
	}

	deputies := claim.DeputyAddresses{}
	for k, v := range temp.Deputies {
		deputies[k] = claim.DeputyAddress{
			Kava: mustDecodeKavaAddress(v.Kava),
			Bnb:  mustDecodeBnbAddress(v.Bnb),
		}
	}

	cfg := Config{
		BnbRPCURL:     temp.BnbRPCURL,
		KavaGrpcURL:   temp.KavaGrpcURL,
		Deputies:      deputies,
		BnbMnemonics:  temp.BnbMnemonics,
		KavaMnemonics: temp.KavaMnemonics,
	}
	return cfg, nil
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
