module github.com/kava-labs/go-tools/auction-alerts

go 1.15

replace github.com/kava-labs/go-tools/slack-alerts v0.1.0 => ../slack-alerts

require (
	github.com/cosmos/cosmos-sdk v0.39.2
	github.com/joho/godotenv v1.3.0
	github.com/kava-labs/go-tools/slack-alerts v0.1.0
	github.com/kava-labs/kava v0.14.0-rc1
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/tendermint v0.33.9
)
