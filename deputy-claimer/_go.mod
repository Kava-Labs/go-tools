module github.com/kava-labs/go-tools/deputy-claimer

go 1.13

require (
	github.com/cosmos/cosmos-sdk v0.44.5
	github.com/golang/mock v1.6.0
	github.com/kava-labs/binance-chain-go-sdk v1.2.5-kava
	github.com/kava-labs/go-sdk v0.5.1-0.20211220154055-4aeeffe85ecd
	github.com/kava-labs/kava v0.15.2-0.20211229145201-28c1167dd41f
	github.com/kava-labs/tendermint v0.32.3-kava1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.14
	google.golang.org/grpc v1.43.0
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
