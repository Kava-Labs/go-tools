package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server/config"
	tmconfig "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
)

type NodeConfig struct {
	AppConfig     *config.Config
	TMConfig      *tmconfig.Config
	PrivValidator *privval.FilePV
	NodeKey       *p2p.NodeKey
	GenesisDoc    types.GenesisDoc
	WhalePrivKey  cryptotypes.PrivKey // TODO store all keys(/mnemonics?) used in gen accounts (could be written out to keyring files)
	// TODO include client.toml?
}

// TODO enable api by default
func NewDefaultNodeConfig(homeDir string) NodeConfig {
	tmCfg := tmconfig.DefaultConfig()
	tmCfg.RootDir = homeDir

	nodePrivKey := ed25519.GenPrivKey()
	nodeKey := &p2p.NodeKey{
		PrivKey: nodePrivKey,
	}

	privVal := privval.GenFilePV(tmCfg.PrivValidatorKeyFile(), tmCfg.PrivValidatorStateFile())

	chainID := "kava-localnet"
	valPubKey, err := cryptocodec.FromTmPubKeyInterface(privVal.Key.PubKey)
	if err != nil {
		panic(err)
	}
	valOperPrivKey := secp256k1.GenPrivKey()
	appGen := DefaultKavaAppGenesis(valPubKey, valOperPrivKey, chainID)

	appState, err := json.MarshalIndent(appGen, "", "  ")
	if err != nil {
		panic(err)
	}

	return NodeConfig{
		AppConfig:     config.DefaultConfig(),
		TMConfig:      tmCfg,
		PrivValidator: privVal,
		NodeKey:       nodeKey,
		GenesisDoc: types.GenesisDoc{
			GenesisTime:     time.Now(),
			ChainID:         chainID,
			InitialHeight:   1,
			ConsensusParams: types.DefaultConsensusParams(),
			Validators:      nil,
			AppHash:         nil,
			AppState:        appState,
		},
		WhalePrivKey: valOperPrivKey,
	}
}

func WriteNodeConfig(cfg NodeConfig) error {

	rootDir := cfg.TMConfig.BaseConfig.RootDir // TODO theres lots of root dirs
	if rootDir == "" {
		return fmt.Errorf("expected valid home directory, got '%s'", rootDir)
	}

	appCfgPath := filepath.Join(rootDir, "config", "app.toml")          // TODO import name?
	if err := os.MkdirAll(filepath.Dir(appCfgPath), 0777); err != nil { // TODO permissions?
		return err
	}
	config.WriteConfigFile(appCfgPath, cfg.AppConfig)

	tmCfgPath := filepath.Join(rootDir, "config", "config.toml") // TODO import name?
	tmconfig.WriteConfigFile(tmCfgPath, cfg.TMConfig)

	if err := os.MkdirAll(filepath.Dir(cfg.TMConfig.PrivValidatorKeyFile()), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.TMConfig.PrivValidatorStateFile()), 0777); err != nil {
		return err
	}
	cfg.PrivValidator.Save()

	if err := os.MkdirAll(filepath.Dir(cfg.TMConfig.NodeKeyFile()), 0777); err != nil {
		return err
	}
	cfg.NodeKey.SaveAs(cfg.TMConfig.NodeKeyFile())

	if err := os.MkdirAll(filepath.Dir(cfg.TMConfig.GenesisFile()), 0777); err != nil {
		return err
	}
	cfg.GenesisDoc.SaveAs(cfg.TMConfig.GenesisFile())

	return nil
}
