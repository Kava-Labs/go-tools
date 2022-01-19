module github.com/kava-labs/go-tools/alerts

go 1.16

require (
	github.com/aws/aws-sdk-go-v2 v1.9.1
	github.com/aws/aws-sdk-go-v2/config v1.6.0
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.1.4
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.4.2
	github.com/cosmos/cosmos-sdk v0.44.5
	github.com/joho/godotenv v1.3.0
	github.com/kava-labs/kava v0.16.0
	github.com/slack-go/slack v0.9.4
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.15
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/tendermint/tendermint => github.com/tendermint/tendermint v0.34.15
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
