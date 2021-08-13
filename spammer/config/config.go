package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/prometheus/common/log"

	"github.com/kava-labs/go-tools/spammer/types"
)

// DefaultConfigPath is the default config path
const DefaultConfigPath = "./config/config.json"
const SwapConfigPath = "./config/config_swap.json"

// Config defines chain connections and mnemonics
type Config struct {
	Mnemonic    string         `toml:"mnemonic" json:"mnemonic"`
	RPCEndpoint string         `toml:"rpc_endpoint" json:"rpc_endpoint"`
	NumAccounts int            `toml:"num_accounts" json:"num_accounts"`
	Messages    types.Messages `toml:"messages" json:"messages"`
}

// NewConfig initializes a new empty config
func NewConfig() *Config {
	return &Config{
		Mnemonic:    "",
		RPCEndpoint: "",
		NumAccounts: 0,
		Messages:    nil,
	}
}

// GetConfig loads and validates a configuration file, returning the Config struct if valid
func GetConfig(cdc *codec.Codec, filePath string) (*Config, error) {
	var config Config

	err := loadConfig(cdc, filePath, &config)
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func loadConfig(cdc *codec.Codec, file string, config *Config) error {
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

	fileContents, _ := ioutil.ReadFile(fpClean)
	err = cdc.UnmarshalJSON([]byte(fileContents), &config)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) validate() error {
	if c.Mnemonic == "" {
		return fmt.Errorf("required field Mnemonic is empty")
	}
	if c.RPCEndpoint == "" {
		return fmt.Errorf("required field RPCEndpoint is empty")
	}
	if c.NumAccounts < 0 {
		return fmt.Errorf("required field NumAccounts must have a positive integer value")
	}
	err := c.Messages.Validate()
	if err != nil {
		return err
	}
	return nil
}
