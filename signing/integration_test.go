//go:build integration

package signing_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/kava-labs/go-tools/testutil"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"

	"github.com/kava-labs/go-tools/signing"
)

func TestMain(m *testing.M) {
	app.SetSDKConfig()
	os.Exit(m.Run())
}

type nodeRunner interface {
	addressProvider

	Init() error
	Start() error
	Stop() error
	Cleanup() error
}

type addressProvider interface {
	RPCAddress() string
	GRPCAddress() string
	APIAddress() string
}

type E2ETestSuite struct {
	suite.Suite

	nodeConfig testutil.NodeConfig
	nodeRunner nodeRunner
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (suite *E2ETestSuite) SetupTest() {

	suite.nodeConfig = testutil.NewDefaultNodeConfig(suite.T().TempDir())

	runner, err := testutil.NewDockerNodeRunner(suite.nodeConfig)
	suite.Require().NoError(err)
	suite.nodeRunner = runner

	err = suite.nodeRunner.Init()
	suite.Require().NoError(err)

	err = suite.nodeRunner.Start()
	suite.Require().NoError(err)
}

func (suite *E2ETestSuite) TearDownTest() {
	err := suite.nodeRunner.Cleanup() // TODO what happens if it never started? / was removed earlier?
	suite.Require().NoError(err)
}

func (suite *E2ETestSuite) TestSendTxs() {

	conn, err := grpc.Dial(suite.nodeRunner.GRPCAddress(), grpc.WithInsecure())
	suite.Require().NoError(err)
	defer conn.Close()

	tmClient := tmservice.NewServiceClient(conn)
	nodeInfoResponse, err := tmClient.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	suite.Require().NoError(err)

	txClient := txtypes.NewServiceClient(conn)
	authClient := authtypes.NewQueryClient(conn)

	signer := signing.NewSigner(
		nodeInfoResponse.DefaultNodeInfo.Network,
		app.MakeEncodingConfig(),
		authClient,
		txClient,
		suite.nodeConfig.WhalePrivKey,
		10, // inflightLimit
	)
	requestQueueSize := 100
	requests := make(chan signing.MsgRequest, requestQueueSize)
	responses, err := signer.Run(requests)
	suite.Require().NoError(err)

	toAddr, err := sdk.AccAddressFromBech32("kava1mq9qxlhze029lm0frzw2xr6hem8c3k9ts54w0w")
	if err != nil {
		panic(err)
	}
	bankClient := banktypes.NewQueryClient(conn)
	resp, err := bankClient.AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{Address: toAddr.String()}) // TODO move into method?
	suite.Require().NoError(err)
	originalBalance := resp.Balances

	msg := banktypes.NewMsgSend(
		suite.nodeConfig.WhalePrivKey.PubKey().Address().Bytes(),
		toAddr,
		sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(1))),
	)
	msgRequest := signing.MsgRequest{
		Msgs:      []sdk.Msg{msg},
		GasLimit:  200000,
		FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(1000))),
	}

	for i := 0; i < requestQueueSize; i++ {
		requests <- msgRequest
	}

	suite.T().Log("waiting for txs to be confirmed")

	timeout := time.After(2 * time.Minute)
	count := 0
	var response signing.MsgResponse
waitLoop:
	for {
		select {
		case response = <-responses:
			if response.Err != nil {
				suite.T().Logf("got err for tx: %v", response.Err)
			}
			count++
			if count >= requestQueueSize {
				break waitLoop
			}
		case <-timeout:
			suite.Fail("timed out waiting for rtx responses")
			break waitLoop
		}
	}

	// Check balance is ok

	resp, err = bankClient.AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{Address: toAddr.String()})
	suite.Require().NoError(err)
	newBalance := resp.Balances

	suite.Equal(
		scalarMul(int64(requestQueueSize), msg.Amount),
		newBalance.Sub(originalBalance),
	)
}

func (suite *E2ETestSuite) TestTxsAreSentGivenUnreliableMempool() {

	conn, err := grpc.Dial(suite.nodeRunner.GRPCAddress(), grpc.WithInsecure())
	suite.Require().NoError(err)
	defer conn.Close()

	tmClient := tmservice.NewServiceClient(conn)
	nodeInfoResponse, err := tmClient.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	suite.Require().NoError(err)

	txClient := txtypes.NewServiceClient(conn)
	authClient := authtypes.NewQueryClient(conn)

	signer := signing.NewSigner(
		nodeInfoResponse.DefaultNodeInfo.Network,
		app.MakeEncodingConfig(),
		authClient,
		txClient,
		suite.nodeConfig.WhalePrivKey,
		10, // inflightLimit
	)
	requestQueueSize := 100
	requests := make(chan signing.MsgRequest, requestQueueSize)
	responses, err := signer.Run(requests)
	suite.Require().NoError(err)

	toAddr, err := sdk.AccAddressFromBech32("kava1mq9qxlhze029lm0frzw2xr6hem8c3k9ts54w0w")
	if err != nil {
		panic(err)
	}
	bankClient := banktypes.NewQueryClient(conn)
	resp, err := bankClient.AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{Address: toAddr.String()}) // TODO move into method?
	suite.Require().NoError(err)
	originalBalance := resp.Balances

	msg := banktypes.NewMsgSend(
		suite.nodeConfig.WhalePrivKey.PubKey().Address().Bytes(),
		toAddr,
		sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(1))),
	)
	msgRequest := signing.MsgRequest{
		Msgs:      []sdk.Msg{msg},
		GasLimit:  200000,
		FeeAmount: sdk.NewCoins(sdk.NewCoin("ukava", sdk.NewInt(1000))),
	}

	fmt.Println("########################## submitting")
	for i := 0; i < requestQueueSize; i++ {
		requests <- msgRequest
	}

	suite.T().Log("######### restarting")
	err = suite.nodeRunner.Stop()
	suite.Require().NoError(err)
	err = suite.nodeRunner.Start()
	suite.Require().NoError(err)

	fmt.Println("########################## waiting in test")

	timeout := time.After(60 * time.Second)
	count := 0
	var response signing.MsgResponse
waitLoop:
	for {
		select {
		case response = <-responses:
			if response.Err != nil {
				suite.T().Logf("got err for tx: %v", response.Err)
			}
			count++
			if count >= requestQueueSize {
				break waitLoop
			}
		case <-timeout:
			fmt.Println("########################## timed out")
			break waitLoop
		}
	}

	// Check balance is ok

	resp, err = bankClient.AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{Address: toAddr.String()})
	suite.Require().NoError(err)
	newBalance := resp.Balances

	suite.Equal(
		scalarMul(int64(requestQueueSize), msg.Amount),
		newBalance.Sub(originalBalance),
	)
}

func scalarMul(scalar int64, coins sdk.Coins) sdk.Coins {
	multiplied := sdk.NewCoins()
	for _, c := range coins {
		multiplied = multiplied.Add(
			sdk.NewCoin(c.Denom, c.Amount.MulRaw(scalar)),
		)
	}
	return multiplied
}
