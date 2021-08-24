package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	kavaRpcUrlEnvKey        = "KAVA_RPC_URL"
	slackTokenEnvKey        = "SLACK_TOKEN"
	slackChannelIdEnvKey    = "SLACK_CHANNEL_ID"
	intervalEnvKey          = "INTERVAL"
	alertFrequencyEnvKey    = "ALERT_FREQUENCY"
	usdThresholdEnvKey      = "USD_THRESHOLD"
	dynamoDbTableNameEnvKey = "DYNAMODB_TABLE_NAME"
)

// ConfigLoader provides an interface for loading config values from a provided
// key
type ConfigLoader interface {
	Get(key string) string
}

// Config provides application configuration
type Config struct {
	KavaRpcUrl string
	// Interval at which the process runs to check ongoing auctions
	Interval       time.Duration
	AlertFrequency time.Duration
	SlackToken     string
	SlackChannelId string
	// US dollar value of auctions that triggers alert
	UsdThreshold      sdk.Dec
	DynamoDbTableName string
}

// LoadConfig loads key values from a ConfigLoader and returns a new Config
func LoadConfig(loader ConfigLoader) (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, err
	}
	rpcURL := loader.Get(kavaRpcUrlEnvKey)
	if rpcURL == "" {
		return Config{}, fmt.Errorf("%s not set", kavaRpcUrlEnvKey)
	}

	dynamoDbTableName := loader.Get(dynamoDbTableNameEnvKey)

	slackToken := loader.Get(slackTokenEnvKey)
	slackChannelId := loader.Get(slackChannelIdEnvKey)
	usdThreshold := loader.Get(usdThresholdEnvKey)

	usdThresholdDec, err := sdk.NewDecFromStr(usdThreshold)
	if err != nil {
		return Config{}, err
	}

	updateInterval, err := time.ParseDuration(loader.Get(intervalEnvKey))
	if err != nil {
		updateInterval = time.Duration(10 * time.Minute)
	}

	alertFrequency, err := time.ParseDuration(loader.Get(alertFrequencyEnvKey))
	if err != nil {
		updateInterval = time.Duration(10 * time.Minute)
	}

	return Config{
		KavaRpcUrl:        rpcURL,
		Interval:          updateInterval,
		AlertFrequency:    alertFrequency,
		SlackToken:        slackToken,
		SlackChannelId:    slackChannelId,
		UsdThreshold:      usdThresholdDec,
		DynamoDbTableName: dynamoDbTableName,
	}, nil
}
