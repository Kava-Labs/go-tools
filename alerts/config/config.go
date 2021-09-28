package config

const (
	kavaRpcUrlEnvKey        = "KAVA_RPC_URL"
	dynamoDbTableNameEnvKey = "DYNAMODB_TABLE_NAME"
	slackTokenEnvKey        = "SLACK_TOKEN"
	slackChannelIdEnvKey    = "SLACK_CHANNEL_ID"
	intervalEnvKey          = "INTERVAL"
	alertFrequencyEnvKey    = "ALERT_FREQUENCY"
	usdThresholdEnvKey      = "USD_THRESHOLD"
	usdxDeviationEnvKey     = "USDX_DEVIATION"
	usdxBasePriceKey        = "USDX_BASE_PRICE"
)

// ConfigLoader provides an interface for loading config values from a provided
// key
type ConfigLoader interface {
	Get(key string) string
}
