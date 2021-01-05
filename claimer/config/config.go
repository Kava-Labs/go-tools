package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

const DefaultConfigPath = "./config/config.json"

// Config defines chain connections and mnemonics
type Config struct {
	Kava         KavaConfig         `toml:"kava_config" json:"kava_config"`
	BinanceChain BinanceChainConfig `toml:"binance_chain_config" json:"binance_chain_config"`
}

// KavaConfig defines information required for Kava blockchain interaction
type KavaConfig struct {
	ChainID   string   `toml:"chain_id" json:"chain_id"`
	Endpoint  string   `toml:"endpoint" json:"endpoint"`
	Mnemonics []string `toml:"mnemonics" json:"mnemonics"`
}

// BinanceChainConfig defines information required for Binance Chain interaction
type BinanceChainConfig struct {
	ChainID  string `toml:"chain_id" json:"chain_id"`
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

// GetConfig loads and validates the default configuration file, returning the Config struct if valid
func GetConfig(filePath string) (*Config, error) {
	var config Config

	err := loadConfig(filePath, &config)
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
	log.Infof("Loading configuration path %s", fpClean)

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
