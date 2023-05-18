package config

import (
	"fmt"

	kjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const DefaultConfigPath = "./config/config.json"

// Config defines chain connections and mnemonics
type Config struct {
	Kava         KavaConfig         `koanf:"kava_config"`
	BinanceChain BinanceChainConfig `koanf:"binance_chain_config"`
}

// KavaConfig defines information required for Kava blockchain interaction
type KavaConfig struct {
	ChainID   string   `koanf:"chain_id"`
	Endpoint  string   `koanf:"endpoint"`
	Mnemonics []string `koanf:"mnemonics"`
}

// BinanceChainConfig defines information required for Binance Chain interaction
type BinanceChainConfig struct {
	ChainID  string `koanf:"chain_id"`
	Endpoint string `koanf:"endpoint"`
	Mnemonic string `koanf:"mnemonic"`
}

// LoadConfig loads and validates a configuration file, returning the Config struct if valid
func LoadConfig(filePath string) (Config, error) {
	var k = koanf.New(".")

	if err := k.Load(file.Provider(filePath), kjson.Parser()); err != nil {
		return Config{}, fmt.Errorf("error loading config: %w", err)
	}

	var config Config
	k.Unmarshal(".", &config)

	if err := config.validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.Kava.ChainID == "" {
		return fmt.Errorf("required field Kava.ChainID is empty")
	}
	if c.Kava.Endpoint == "" {
		return fmt.Errorf("required field Kava.Endpoint is empty")
	}
	if len(c.Kava.Mnemonics) == 0 {
		return fmt.Errorf("required field Kava.Mnemonics is empty")
	}
	if c.BinanceChain.Endpoint == "" {
		return fmt.Errorf("required field BinanceChain.Endpoint is empty")
	}
	if c.BinanceChain.Mnemonic == "" {
		return fmt.Errorf("required field BinanceChain.Mnemonic is empty")
	}
	if c.BinanceChain.ChainID == "" {
		return fmt.Errorf("required field BinanceChain.ChainID is empty")
	}
	return nil
}
