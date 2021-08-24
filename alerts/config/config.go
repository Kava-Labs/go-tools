package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	kavaRpcUrlEnvKey        = "KAVA_RPC_URL"
	dynamoDbTableNameEnvKey = "DYNAMODB_TABLE_NAME"
	slackTokenEnvKey        = "SLACK_TOKEN"
	slackChannelIdEnvKey    = "SLACK_CHANNEL_ID"
	intervalEnvKey          = "INTERVAL"
	alertFrequencyEnvKey    = "ALERT_FREQUENCY"
	usdThresholdEnvKey      = "USD_THRESHOLD"
)

// ConfigLoader provides an interface for loading config values from a provided
// key
type ConfigLoader interface {
	Get(key string) string
}

// Config provides application configuration
type BaseConfig struct {
	KavaRpcUrl        string
	DynamoDbTableName string
	SlackToken        string
	SlackChannelId    string
	Interval          time.Duration
	AlertFrequency    time.Duration
}

type AuctionsConfig struct {
	BaseConfig
	// US dollar value of auctions that triggers alert
	UsdThreshold sdk.Dec
}

// LoadAuctionsConfig loads key values from a ConfigLoader and returns a new AuctionsConfig
func LoadAuctionsConfig(loader ConfigLoader) (AuctionsConfig, error) {
	baseConfig, err := LoadBaseConfig(loader)
	if err != nil {
		return AuctionsConfig{}, err
	}

	usdThreshold := loader.Get(usdThresholdEnvKey)

	usdThresholdDec, err := sdk.NewDecFromStr(usdThreshold)
	if err != nil {
		return AuctionsConfig{}, err
	}

	return AuctionsConfig{
		BaseConfig:   baseConfig,
		UsdThreshold: usdThresholdDec,
	}, nil
}

// LoadBaseConfig loads key values from a ConfigLoader and returns a new
// BaseConfig used for multiple different commands
func LoadBaseConfig(loader ConfigLoader) (BaseConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return BaseConfig{}, err
	}
	rpcURL := loader.Get(kavaRpcUrlEnvKey)
	if rpcURL == "" {
		return BaseConfig{}, fmt.Errorf("%s not set", kavaRpcUrlEnvKey)
	}

	dynamoDbTableName := loader.Get(dynamoDbTableNameEnvKey)

	slackToken := loader.Get(slackTokenEnvKey)
	slackChannelId := loader.Get(slackChannelIdEnvKey)

	updateInterval, err := time.ParseDuration(loader.Get(intervalEnvKey))
	if err != nil {
		updateInterval = time.Duration(10 * time.Minute)
	}

	alertFrequency, err := time.ParseDuration(loader.Get(alertFrequencyEnvKey))
	if err != nil {
		updateInterval = time.Duration(10 * time.Minute)
	}

	return BaseConfig{
		KavaRpcUrl:        rpcURL,
		Interval:          updateInterval,
		AlertFrequency:    alertFrequency,
		SlackToken:        slackToken,
		SlackChannelId:    slackChannelId,
		DynamoDbTableName: dynamoDbTableName,
	}, nil
}
