package bank2

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/testutil/constants"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func init() {
	if err := evmtypes.NewEVMConfigurator().
		WithEVMCoinInfo(constants.ExampleChainCoinInfo[constants.ExampleChainID]).
		Configure(); err != nil {
		panic(err)
	}
}

func TestERC20ContractAddress(t *testing.T) {
	denom := "uatom"
	contract := common.HexToAddress(evmtypes.Bank2PrecompileAddress)
	expected := common.HexToAddress("0xdDe94B5b492d597317FD86d2A5baad9966BE2e3e")

	result := ERC20ContractAddress(contract, denom)
	require.Equal(t, expected, result)
}

func Setup() *vm.EVM {
	rawdb := dbm.NewMemDB()
	logger := log.NewNopLogger()
	ms := store.NewCommitMultiStore(rawdb, logger, nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{}, false, logger)
	return NewMockEVM(ctx)
}

func TestBankPrecompile(t *testing.T) {
	evm := Setup()

	bankKeeper := NewMockBankKeeper()
	msgServer := NewBankMsgServer(bankKeeper)
	precompile := NewPrecompile(msgServer, bankKeeper)
	evm.WithPrecompiles(map[common.Address]vm.PrecompiledContract{
		precompile.Address(): precompile,
	})

	denom := "aatom"
	symbol := "ATOM"
	name := "Cosmos Hub Atom"
	decimals := byte(18)
	bankKeeper.registerDenom(denom, banktypes.Metadata{
		Symbol: symbol, Name: name, DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: uint32(decimals),
			},
		},
	})

	testCases := []struct {
		name   string
		method BankMethod
		input  []byte
		output []byte
	}{
		{"name", MethodName, []byte(denom), []byte(name)},
		{"symbol", MethodSymbol, []byte(denom), []byte(symbol)},
		{"decimals", MethodDecimals, []byte(denom), []byte{decimals}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := append([]byte{byte(MethodName)}, denom...)
			ret, _, err := evm.Call(common.Address{}, precompile.Address(), input, 1000000, uint256.NewInt(0))
			require.NoError(t, err)
			require.Equal(t, []byte(name), ret)
		})
	}
}
