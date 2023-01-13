package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	grpcUrlEnvKey = "GRPC_URL"
	rpcUrlEnvKey  = "RPC_URL"
	startEnvKey   = "START_HEIGHT"
	endEnvKey     = "END_HEIGHT"
)

// ConfigLoader provides an interface for
// loading config values from a provided key
type ConfigLoader interface {
	Get(key string) string
}

// EnvLoader loads keys from os environment
type EnvLoader struct{}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

type Config struct {
	GrpcURL string
	RpcURL  string

	StartHeight int64
	EndHeight   int64
}

func LoadConfig(loader ConfigLoader) (Config, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf(".env not found, attempting to proceed with available env variables\n")
	}

	grpcURL := loader.Get(grpcUrlEnvKey)
	if grpcURL == "" {
		return Config{}, fmt.Errorf("%s not set", grpcUrlEnvKey)
	}

	rpcURL := loader.Get(rpcUrlEnvKey)
	if rpcURL == "" {
		return Config{}, fmt.Errorf("%s not set", rpcUrlEnvKey)
	}

	startHeightStr := loader.Get(startEnvKey)
	startHeight, err := strconv.ParseInt(startHeightStr, 10, 64)
	if err != nil {
		return Config{}, err
	}

	endHeightStr := loader.Get(endEnvKey)
	endHeight, err := strconv.ParseInt(endHeightStr, 10, 64)
	if err != nil {
		return Config{}, err
	}

	return Config{
		GrpcURL:     grpcURL,
		RpcURL:      rpcURL,
		StartHeight: startHeight,
		EndHeight:   endHeight,
	}, nil
}
