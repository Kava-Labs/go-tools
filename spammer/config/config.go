package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/prometheus/common/log"
)

const DefaultConfigPath = "./config/config.json"

// Config defines chain connections and mnemonics
type Config struct {
	Mnemonic    string `toml:"mnemonic" json:"mnemonic"`
	RPCEndpoint string `toml:"rpc_endpoint" json:"rpc_endpoint"`
	NumAccounts int    `toml:"num_accounts" json:"num_accounts"`
}

// NewConfig initializes a new empty config
func NewConfig() *Config {
	return &Config{
		Mnemonic:    "",
		RPCEndpoint: "",
		NumAccounts: 0,
	}
}

// GetConfig loads and validates a configuration file, returning the Config struct if valid
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
	if c.Mnemonic == "" {
		return fmt.Errorf("required field Core.Mnemonic is empty")
	}
	if c.RPCEndpoint == "" {
		return fmt.Errorf("required field Core.RPCEndpoint is empty")
	}
	if c.NumAccounts < 0 {
		return fmt.Errorf("field Core.NumAccounts must have a positive integer value")
	}
	return nil
}
