module github.com/kava-labs/go-tools/auction-bot

go 1.21.9

require (
	github.com/cometbft/cometbft v0.37.4
	github.com/cosmos/cosmos-sdk v0.47.10
	github.com/gorilla/handlers v1.5.1
	github.com/gorilla/mux v1.8.0
	github.com/kava-labs/kava v0.26.1
	github.com/stretchr/testify v1.9.0
	golang.org/x/sync v0.4.0
	github.com/gogo/protobuf v1.3.2
	google.golang.org/grpc v1.63.2
	github.com/btcsuite/btcd v0.23.4
	github.com/btcsuite/btcd/btcec/v2 v2.3.2
)

replace (
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	// Use the cosmos keyring code
	github.com/99designs/keyring => github.com/cosmos/keyring v1.2.0
	// Use cometbft fork of tendermint
	github.com/cometbft/cometbft => github.com/kava-labs/cometbft v0.37.4-kava.1
	github.com/cometbft/cometbft-db => github.com/kava-labs/cometbft-db v0.9.1-kava.1
	// Use cosmos-sdk fork with backported fix for unsafe-reset-all, staking transfer events, and custom tally handler support
	github.com/cosmos/cosmos-sdk => github.com/kava-labs/cosmos-sdk v0.47.10-kava.1
	// See https://github.com/cosmos/cosmos-sdk/pull/13093
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt/v4 v4.4.2
	// Use ethermint fork that respects min-gas-price with NoBaseFee true and london enabled, and includes eip712 support
	github.com/evmos/ethermint => github.com/kava-labs/ethermint v0.21.0-kava-v26.2
	// Downgraded to avoid bugs in following commits which causes "version does not exist" errors
	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
	// stick with compatible version or x/exp in v0.47.x line
	golang.org/x/exp => golang.org/x/exp v0.0.0-20230711153332-06a737ee72cb
	// stick with compatible version of rapid in v0.47.x line
	pgregory.net/rapid => pgregory.net/rapid v0.5.5
)
