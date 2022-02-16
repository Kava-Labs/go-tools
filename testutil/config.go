package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cosmos/cosmos-sdk/server/config"
	"github.com/kava-labs/kava/app"
	tmconfig "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type NodeConfigBuilder struct {
	AppConfig     *config.Config
	TMConfig      *tmconfig.Config
	PrivValidator *privval.FilePV
	NodeKey       *p2p.NodeKey
	GenesisDoc    types.GenesisDoc
	// TODO include client keys?
}

func NewDefaultNodeConfig(homeDir string) *NodeConfigBuilder {
	tmCfg := tmconfig.DefaultConfig()
	tmCfg.RootDir = homeDir

	nodePrivKey := ed25519.GenPrivKey()
	nodeKey := &p2p.NodeKey{
		PrivKey: nodePrivKey,
	}

	appState, err := json.MarshalIndent(app.NewDefaultGenesisState(), "", "  ")
	if err != nil {
		panic(err)
	}

	return &NodeConfigBuilder{
		AppConfig:     config.DefaultConfig(),
		TMConfig:      tmCfg,
		PrivValidator: privval.GenFilePV(tmCfg.PrivValidatorKeyFile(), tmCfg.PrivValidatorStateFile()),
		NodeKey:       nodeKey,
		GenesisDoc: types.GenesisDoc{
			GenesisTime:     time.Now(),
			ChainID:         "kava-localnet",
			InitialHeight:   1,
			ConsensusParams: types.DefaultConsensusParams(),
			Validators:      nil,
			AppHash:         nil,
			AppState:        appState,
		},
	}
}

/*
TODO Genesis setup
- change denoms to kava
- add account and coins
- add validator gentx
*/

func WriteNodeConfig(
	appConfig *config.Config,
	tmConfig *tmconfig.Config,
	privValidator *privval.FilePV,
	nodeKey *p2p.NodeKey,
	genesisDoc tmtypes.GenesisDoc,
) error {

	rootDir := tmConfig.BaseConfig.RootDir // TODO theres lots of root dirs
	if rootDir == "" {
		return fmt.Errorf("expected valid home directory, got '%s'", rootDir)
	}

	appCfgPath := filepath.Join(rootDir, "config", "app.toml")          // TODO import name?
	if err := os.MkdirAll(filepath.Dir(appCfgPath), 0777); err != nil { // TODO permissions?
		return err
	}
	config.WriteConfigFile(appCfgPath, appConfig)

	tmCfgPath := filepath.Join(rootDir, "config", "config.toml") // TODO import name?
	tmconfig.WriteConfigFile(tmCfgPath, tmConfig)

	if err := os.MkdirAll(filepath.Dir(tmConfig.PrivValidatorKeyFile()), 0777); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(tmConfig.PrivValidatorStateFile()), 0777); err != nil {
		return err
	}
	privValidator.Save()

	if err := os.MkdirAll(filepath.Dir(tmConfig.NodeKeyFile()), 0777); err != nil {
		return err
	}
	nodeKey.SaveAs(tmConfig.NodeKeyFile())

	if err := os.MkdirAll(filepath.Dir(tmConfig.GenesisFile()), 0777); err != nil {
		return err
	}
	genesisDoc.SaveAs(tmConfig.GenesisFile())

	return nil
}
