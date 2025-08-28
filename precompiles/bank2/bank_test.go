package bank2

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
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
	user1 := common.BigToAddress(big.NewInt(1))
	user2 := common.BigToAddress(big.NewInt(2))

	denom := "denom"
	symbol := "COIN"
	name := "Test Coin"
	decimals := byte(18)
	amount := int64(1000)
	precompile := common.HexToAddress(evmtypes.Bank2PrecompileAddress)
	erc20 := ERC20ContractAddress(precompile, denom)

	setup := func(t *testing.T) *vm.EVM {
		evm := Setup()

		bankKeeper := NewMockBankKeeper()
		msgServer := NewBankMsgServer(bankKeeper)
		precompile := NewPrecompile(msgServer, bankKeeper)
		evm.WithPrecompiles(map[common.Address]vm.PrecompiledContract{
			precompile.Address(): precompile,
		})

		bankKeeper.registerDenom(denom, banktypes.Metadata{
			Symbol: symbol, Name: name, DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    denom,
					Exponent: uint32(decimals),
				},
			},
		})
		bankKeeper.mint(user1.Bytes(), sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(amount))))

		return evm
	}

	testCases := []struct {
		name   string
		method BankMethod
		expErr error
		caller common.Address
		input  []byte
		output []byte
	}{
		{"name", MethodName, nil, user1, []byte(denom), []byte(name)},
		{"symbol", MethodSymbol, nil, user1, []byte(denom), []byte(symbol)},
		{"decimals", MethodDecimals, nil, user1, []byte(denom), []byte{decimals}},
		{"totalSupply", MethodTotalSupply, nil, user1, []byte(denom),
			common.LeftPadBytes(big.NewInt(amount).Bytes(), 32),
		},
		{"balanceOf", MethodBalanceOf, nil, user1,
			append(user1.Bytes(), []byte(denom)...),
			common.LeftPadBytes(big.NewInt(amount).Bytes(), 32),
		},
		{"balanceOf-empty", MethodBalanceOf, nil, user2,
			append(user2.Bytes(), []byte(denom)...),
			common.LeftPadBytes([]byte{}, 32),
		},
		{"transferFrom-owner", MethodTransferFrom, nil, user1,
			transferFromInput(user1, user2, big.NewInt(100), denom),
			[]byte{1},
		},
		{"transferFrom-erc20", MethodTransferFrom, nil, erc20,
			transferFromInput(user1, user2, big.NewInt(100), denom),
			[]byte{1},
		},
		{"transferFrom-unauthorized", MethodTransferFrom, vm.ErrExecutionReverted, user2,
			transferFromInput(user1, user2, big.NewInt(100), denom),
			nil,
		},
		{"transferFrom-insufficient-balance", MethodTransferFrom, vm.ErrExecutionReverted, user2,
			transferFromInput(user2, user1, big.NewInt(100), denom),
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evm := setup(t)
			input := append([]byte{byte(tc.method)}, tc.input...)
			ret, _, err := evm.Call(tc.caller, precompile, input, 1000000, uint256.NewInt(0))
			if tc.expErr != nil {
				require.Equal(t, tc.expErr, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.output, ret)
			}
		})
	}
}

func transferFromInput(from, to common.Address, amount *big.Int, denom string) []byte {
	fixedBuf := make([]byte, 20+20+32) // from + to + amount
	copy(fixedBuf[0:20], from.Bytes())
	copy(fixedBuf[20:40], to.Bytes())
	copy(fixedBuf[40:72], common.LeftPadBytes(amount.Bytes(), 32))
	return append(fixedBuf, []byte(denom)...)
}
