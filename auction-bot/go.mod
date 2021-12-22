module github.com/kava-labs/go-tools/auction-bot

go 1.15

require (
	github.com/cosmos/cosmos-sdk v0.44.5
	github.com/joho/godotenv v1.3.0
	github.com/kava-labs/kava v0.15.2-0.20211221173220-559df6414b46
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.14
	google.golang.org/grpc v1.43.0 // indirect
)

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
