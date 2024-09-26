package config

import (
	"fmt"
)

// GrpcConfig provides Kava GRPC configuration
type GrpcConfig struct {
	KavaGrpcUrl string
}

// LoadGrpcConfig loads key values from a ConfigLoader and returns a new
// GrpcConfig used for multiple different commands
func LoadGrpcConfig(loader ConfigLoader) (GrpcConfig, error) {
	rpcURL := loader.Get(kavaGrpcUrlEnvKey)
	if rpcURL == "" {
		return GrpcConfig{}, fmt.Errorf("%s not set", kavaGrpcUrlEnvKey)
	}

	return GrpcConfig{
		KavaGrpcUrl: rpcURL,
	}, nil
}
