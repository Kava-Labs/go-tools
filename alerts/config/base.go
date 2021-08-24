package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

// Config provides application configuration
type BaseConfig struct {
	KavaRpcUrl        string
	DynamoDbTableName string
	SlackToken        string
	SlackChannelId    string
	Interval          time.Duration
	AlertFrequency    time.Duration
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
