//go:build integration

package signing_test

import (
	"os"
	"testing"

	"github.com/kava-labs/go-tools/testutil"
	"github.com/kava-labs/kava/app"
	"github.com/stretchr/testify/suite"
)

func TestMain(m *testing.M) {
	app.SetSDKConfig()
	os.Exit(m.Run())
}

type nodeRunner interface {
	Start() error
	Stop() error
	Cleanup() error
}

type E2ETestSuite struct {
	suite.Suite

	nodeRunner nodeRunner
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (suite *E2ETestSuite) SetupTest() {

	cfg := testutil.NewDefaultNodeConfig(suite.T().TempDir())

	runner, err := testutil.NewDockerNodeRunner(
		cfg.AppConfig,
		cfg.TMConfig,
		cfg.PrivValidator,
		cfg.NodeKey,
		cfg.GenesisDoc,
	)

	suite.Require().NoError(err)
	suite.nodeRunner = runner

	err = suite.nodeRunner.Start()
	suite.Require().NoError(err)
}

func (suite *E2ETestSuite) TearDownTest() {
	err := suite.nodeRunner.Cleanup() // TODO what happens if it never started? / was removed earlier?
	suite.Require().NoError(err)
}

func (suite *E2ETestSuite) TestTest() {

}
