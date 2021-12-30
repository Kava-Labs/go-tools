package main

import (
	"os"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/crypto"
)

type testEnvLoader struct {
	t   *testing.T
	Env map[string]string
}

func (l *testEnvLoader) Get(key string) string {
	value, ok := l.Env[key]

	if !ok {
		return ""
	}

	return value
}

func TestConfigLoading(t *testing.T) {
	// loader that implements Getenv with test configuration data
	loader := &testEnvLoader{
		t: t,
		Env: map[string]string{
			"KAVA_RPC_URL":         "https://rpc.testnet.kava.io:443",
			"KAVA_GRPC_URL":        "https://grpc.testnet.kava.io:443",
			"KAVA_KEEPER_ADDRESS":  sdk.AccAddress(crypto.AddressHash([]byte("keeper"))).String(),
			"KAVA_SIGNER_MNEMONIC": "arrive guide way exit polar print kitchen hair series custom siege afraid shrug crew fashion mind script divorce pattern trust project regular robust safe",
		},
	}

	defaultConfig, err := LoadConfig(loader)
	if err != nil {
		t.Fatalf("failed with err %s", err)
	}

	if defaultConfig.KavaRpcUrl != loader.Env["KAVA_RPC_URL"] {
		t.Fatalf("bad value %s for KavaRpcUrl", defaultConfig.KavaRpcUrl)
	}

	if defaultConfig.KavaGrpcUrl != loader.Env["KAVA_GRPC_URL"] {
		t.Fatalf("bad value %s for KavaGrpcUrl", defaultConfig.KavaGrpcUrl)
	}

	if defaultConfig.KavaLiquidationInterval != time.Duration(10*time.Minute) {
		t.Fatalf("default liquidation interval is not 10m")
	}

	if defaultConfig.KavaKeeperAddress.String() != loader.Env["KAVA_KEEPER_ADDRESS"] {
		t.Fatalf("bad value %s for KavaKeeperAddress", defaultConfig.KavaKeeperAddress)
	}

	if defaultConfig.KavaSignerMnemonic != loader.Env["KAVA_SIGNER_MNEMONIC"] {
		t.Fatalf("bad value %s for KavaSignerMnemonic", defaultConfig.KavaSignerMnemonic)
	}

	loader = &testEnvLoader{
		t: t,
		Env: map[string]string{
			"KAVA_RPC_URL":              "https://rpc.testnet.kava.io:443",
			"KAVA_GRPC_URL":             "https://grpc.testnet.kava.io:443",
			"KAVA_KEEPER_ADDRESS":       sdk.AccAddress(crypto.AddressHash([]byte("keeper"))).String(),
			"KAVA_LIQUIDATION_INTERVAL": "30m",
			"KAVA_SIGNER_MNEMONIC":      "arrive guide way exit polar print kitchen hair series custom siege afraid shrug crew fashion mind script divorce pattern trust project regular robust safe",
		},
	}

	config, err := LoadConfig(loader)
	if err != nil {
		t.Fatalf("failed with err %s", err)
	}

	if config.KavaLiquidationInterval != time.Duration(30*time.Minute) {
		t.Fatalf("default liquidation interval is not 30m")
	}
}

func TestInvalidKeeperAddress(t *testing.T) {
	loader := &testEnvLoader{
		t: t,
		Env: map[string]string{
			"KAVA_RPC_URL":         "https://rpc.testnet.kava.io:443",
			"KAVA_GRPC_URL":        "https://grpc.testnet.kava.io:443",
			"KAVA_KEEPER_ADDRESS":  "kava1invalidaddress",
			"KAVA_SIGNER_MNEMONIC": "arrive guide way exit polar print kitchen hair series custom siege afraid shrug crew fashion mind script divorce pattern trust project regular robust safe",
		},
	}

	_, err := LoadConfig(loader)
	assert.NotNil(t, err)
	assert.Regexp(t, "decoding bech32 failed", err.Error())
}

func TestEnvLoader(t *testing.T) {
	testKey := "KAVA_CONFIG_VAR_TEST_1"
	testValue := "KAVA_CONFIG_VAR_TEST_1 value test"

	old := os.Getenv(testKey)
	os.Setenv(testKey, testValue)
	defer os.Setenv(testKey, old)

	loader := &EnvLoader{}

	if loader.Get(testKey) != testValue {
		t.Fatalf("config value %s for %s does not match", testValue, testKey)
	}
}
