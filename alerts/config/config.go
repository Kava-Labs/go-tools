package config

const (
	kavaRpcUrlEnvKey               = "KAVA_RPC_URL"
	dynamoDbTableNameEnvKey        = "DYNAMODB_TABLE_NAME"
	slackTokenEnvKey               = "SLACK_TOKEN"
	slackChannelIdEnvKey           = "SLACK_CHANNEL_ID"
	intervalEnvKey                 = "INTERVAL"
	alertFrequencyEnvKey           = "ALERT_FREQUENCY"
	usdThresholdEnvKey             = "USD_THRESHOLD"
	inefficientThresholdEnvKey     = "INEFFICIENT_AUCTION_USD_THRESHOLD"  // the USD of the auction lot for the alerter to care about it -> If an auction's lot is above the threshold, the alert can be triggered
	inefficientRatioEnvKey         = "INEFFICIENT_AUCTION_RATIO"          // the ratio of bid:lot in USD. Below this ratio, the alert can be triggered
	inefficientTimeRemainingEnvKey = "INEFFICIENT_AUCTION_TIME_THRESHOLD" // the amount of time remaining in the auction. If below this duration, the alert can be triggered
	usdxDeviationEnvKey            = "USDX_DEVIATION"
	usdxBasePriceKey               = "USDX_BASE_PRICE"
)

// ConfigLoader provides an interface for loading config values from a provided
// key
type ConfigLoader interface {
	Get(key string) string
}
