package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// ConfigLoader provides an interface for
// loading config values from a provided key
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
	// Spread percent threshold that triggers alerts
	SpreadPercentThreshold float64
	DynamoDbTableName      string
}

// LoadConfig loads key values from a ConfigLoader
// and returns a new Config
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
	spreadThresholdPercent := loader.Get(spreadPercentThresholdEnvKey)

	spreadThreshold, err := strconv.ParseFloat(spreadThresholdPercent, 64)
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
		KavaRpcUrl:             rpcURL,
		Interval:               updateInterval,
		AlertFrequency:         alertFrequency,
		SlackToken:             slackToken,
		SlackChannelId:         slackChannelId,
		SpreadPercentThreshold: spreadThreshold,
		DynamoDbTableName:      dynamoDbTableName,
	}, nil
}

// EnvLoader loads keys from os environment
type EnvLoader struct {
}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}
