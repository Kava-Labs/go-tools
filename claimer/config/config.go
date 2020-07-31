package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const DefaultConfigPath = "./config/config.json"

// Config ...
type Config struct {
	Kava         KavaConfig         `toml:"kava_config" json:"kava_config"`
	BinanceChain BinanceChainConfig `toml:"binance_chain_config" json:"binance_chain_config"`
}

type KavaConfig struct {
	Endpoint  string   `toml:"endpoint" json:"endpoint"`
	Mnemonics []string `toml:"mnemonics" json:"mnemonics"`
}

type BinanceChainConfig struct {
	Endpoint string `toml:"endpoint" json:"endpoint"`
	Mnemonic string `toml:"mnemonic" json:"mnemonic"`
}

// NewConfig initializes a new config
func NewConfig() *Config {
	return &Config{
		Kava:         KavaConfig{},
		BinanceChain: BinanceChainConfig{},
	}
}

func GetConfig() (*Config, error) {
	var config Config

	err := loadConfig(DefaultConfigPath, &config)
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func loadConfig(file string, config *Config) error {
	ext := filepath.Ext(file)
	if ext != ".json" {
		return fmt.Errorf("config file extention must be .json")
	}

	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}

	fpClean := filepath.Clean(fp)
	fmt.Println("Loading configuration", "path", fpClean)

	f, err := os.Open(fpClean)
	if err != nil {
		return err
	}

	if err = json.NewDecoder(f).Decode(&config); err != nil {
		return err
	}

	return nil
}

func (c *Config) validate() error {
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
	return nil
}
