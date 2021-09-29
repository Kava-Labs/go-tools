package config

import (
	"fmt"
)

// LoadRpcConfig provides Kava RPC configuration
type RpcConfig struct {
	KavaRpcUrl string
}

// LoadRpcConfig loads key values from a ConfigLoader and returns a new
// RpcConfig used for multiple different commands
func LoadRpcConfig(loader ConfigLoader) (RpcConfig, error) {
	rpcURL := loader.Get(kavaRpcUrlEnvKey)
	if rpcURL == "" {
		return RpcConfig{}, fmt.Errorf("%s not set", kavaRpcUrlEnvKey)
	}

	return RpcConfig{
		KavaRpcUrl: rpcURL,
	}, nil
}
