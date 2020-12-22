module github.com/kava-labs/go-tools/claimer

go 1.14

require (
	github.com/cosmos/cosmos-sdk v0.39.1
	github.com/gorilla/mux v1.7.4
	github.com/kava-labs/binance-chain-go-sdk v1.2.5-kava
	github.com/kava-labs/go-sdk v0.3.0-rc1
	github.com/kava-labs/go-tools/deputy-claimer v0.0.0-00010101000000-000000000000
	github.com/kava-labs/kava v0.11.0-rc1
	github.com/kava-labs/tendermint v0.32.3-kava1
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.6.1
	github.com/tendermint/go-amino v0.15.1
	github.com/tendermint/tendermint v0.33.7
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208
)

replace github.com/kava-labs/go-tools/deputy-claimer => ../deputy-claimer
