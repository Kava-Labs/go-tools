module github.com/kava-labs/go-tools/claimer

go 1.16

require (
	github.com/cosmos/cosmos-sdk v0.44.5
	github.com/gorilla/mux v1.8.0
	github.com/kava-labs/binance-chain-go-sdk v1.2.5-kava
	github.com/kava-labs/go-sdk v0.5.1-0.20211220154055-4aeeffe85ecd // indirect
	github.com/kava-labs/go-tools/deputy-claimer v0.0.0-20201223131958-31f436616e96
	github.com/kava-labs/kava v0.15.2-0.20211214221953-de3c74c88e69
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/tendermint/tendermint v0.34.14
	golang.org/x/sys v0.0.0-20211210111614-af8b64212486 // indirect
)

replace github.com/kava-labs/go-tools/deputy-claimer => ../deputy-claimer

replace github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
