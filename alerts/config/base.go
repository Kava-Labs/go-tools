package config

import (
	"fmt"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	"github.com/joho/godotenv"
)

// Config provides application configuration
type BaseConfig struct {
	DynamoDbTableName string
	SlackWebhookUrl   string
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

	slackWebhookUrl := loader.Get(slackWebhookUrlEnvKey)
	if slackWebhookUrl == "" {
		return BaseConfig{}, fmt.Errorf("%s not set", slackWebhookUrlEnvKey)
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
		SlackWebhookUrl:   slackWebhookUrl,
		DynamoDbTableName: dynamoDbTableName,
	}, nil
}
