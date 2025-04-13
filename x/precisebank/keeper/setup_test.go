package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/cosmos/evm/testutil/integration/os/network"
	"github.com/cosmos/evm/x/precisebank/types"
)

type KeeperIntegrationTestSuite struct {
	suite.Suite

	network     *network.UnitTestNetwork
	queryClient types.QueryClient
}

func TestKeeperIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperIntegrationTestSuite))
}

func (suite *KeeperIntegrationTestSuite) SetupTest() {
	nw := network.NewUnitTestNetwork()
	suite.network = nw
	suite.queryClient = nw.GetPreciseBankClient()
}
