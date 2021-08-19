module github.com/kava-labs/go-tools/auction-alerts

go 1.15

replace github.com/kava-labs/go-tools/slack-alerts v0.1.0 => ../slack-alerts

require (
	github.com/aws/aws-sdk-go-v2 v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.1.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.4.2 // indirect
	github.com/cosmos/cosmos-sdk v0.39.2
	github.com/joho/godotenv v1.3.0
	github.com/kava-labs/go-tools/slack-alerts v0.1.0
	github.com/kava-labs/kava v0.14.0-rc1
	github.com/tendermint/tendermint v0.33.9
)
