package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/tendermint/tendermint/libs/log"
)

// Config provides application configuration
type BaseConfig struct {
	DynamoDbTableName string
	SlackToken        string
	SlackChannelId    string
	Interval          time.Duration
	AlertFrequency    time.Duration
}

// LoadBaseConfig loads key values from a ConfigLoader and returns a new
// BaseConfig used for multiple different commands
func LoadBaseConfig(loader ConfigLoader, logger log.Logger) (BaseConfig, error) {
	// Ignore error from godotenv, continue if there isn't an .env file and
	// check if required env vars already exist
	if err := godotenv.Load(); err != nil {
		logger.Info(".env not found, attempting to proceed with available env variables")
	}

	dynamoDbTableName := loader.Get(dynamoDbTableNameEnvKey)
	if dynamoDbTableName == "" {
		return BaseConfig{}, fmt.Errorf("%s not set", dynamoDbTableNameEnvKey)
	}

	slackToken := loader.Get(slackTokenEnvKey)
	if slackToken == "" {
		return BaseConfig{}, fmt.Errorf("%s not set", slackToken)
	}

	slackChannelId := loader.Get(slackChannelIdEnvKey)
	if slackChannelId == "" {
		return BaseConfig{}, fmt.Errorf("%s not set", slackChannelId)
	}

	updateInterval, err := time.ParseDuration(loader.Get(intervalEnvKey))
	if err != nil {
		updateInterval = time.Duration(10 * time.Minute)
	}

	alertFrequency, err := time.ParseDuration(loader.Get(alertFrequencyEnvKey))
	if err != nil {
		updateInterval = time.Duration(10 * time.Minute)
	}

	return BaseConfig{
		Interval:          updateInterval,
		AlertFrequency:    alertFrequency,
		SlackToken:        slackToken,
		SlackChannelId:    slackChannelId,
		DynamoDbTableName: dynamoDbTableName,
	}, nil
}
