package bank2

import (
	"testing"

	"github.com/stretchr/testify/require"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
)

func TestERC20ContractAddress(t *testing.T) {
	denom := "uatom"
	contract := common.HexToAddress(evmtypes.Bank2PrecompileAddress)
	expected := common.HexToAddress("0x3f9f3cA556029aECCCFdb6Dd3774D39c10A56b62")

	result := ERC20ContractAddress(contract, denom)
	require.Equal(t, expected, result)
}
