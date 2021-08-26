package config

const (
	kavaRpcUrlEnvKey        = "KAVA_RPC_URL"
	dynamoDbTableNameEnvKey = "DYNAMODB_TABLE_NAME"
	slackTokenEnvKey        = "SLACK_TOKEN"
	slackChannelIdEnvKey    = "SLACK_CHANNEL_ID"
	intervalEnvKey          = "INTERVAL"
	alertFrequencyEnvKey    = "ALERT_FREQUENCY"
)

// ConfigLoader provides an interface for loading config values from a provided
// key
type ConfigLoader interface {
	Get(key string) string
}
