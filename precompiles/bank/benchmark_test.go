package bank

import (
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/testutil/constants"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

var (
	BankPrecompile = common.HexToAddress(evmtypes.BankPrecompileAddress)

	MintTo       = common.BigToAddress(big.NewInt(10000001))
	Erc20Address = common.BigToAddress(big.NewInt(10000002))
)

type TokenInfo struct {
	Denom        string
	DisplayDenom string
	Name         string
	Symbol       string
	Decimals     byte
}

func Setup(b *testing.B, token TokenInfo, mintTo common.Address, mintAmount uint64) *vm.EVM {
	b.Helper()

	chainID := uint64(constants.EighteenDecimalsChainID)
	configurator := evmtypes.NewEVMConfigurator()
	configurator.ResetTestConfig()
	// set global chain config
	ethCfg := evmtypes.DefaultChainConfig(chainID)
	if err := evmtypes.SetChainConfig(ethCfg); err != nil {
		panic(err)
	}
	err := configurator.
		WithExtendedEips(evmtypes.DefaultCosmosEVMActivators).
		// NOTE: we're using the 18 decimals default for the example chain
		WithEVMCoinInfo(constants.ChainsCoinInfo[chainID]).
		Configure()
	require.NoError(b, err)

	// Benchmark implementation would go here
	rawdb := dbm.NewMemDB()
	logger := log.NewNopLogger()
	ms := store.NewCommitMultiStore(rawdb, logger, nil)
	ctx := sdk.NewContext(ms, cmtproto.Header{}, false, logger)
	evm := NewMockEVM(ctx)
	nativeDenom := evmtypes.GetEVMCoinDenom()

	bankKeeper := NewMockBankKeeper()
	erc20keeper := NewMockERC20Keeper()
	precompile := NewPrecompile(bankKeeper, erc20keeper)

	evm.WithPrecompiles(map[common.Address]vm.PrecompiledContract{
		BankPrecompile: precompile,
	})

	// init token
	bankKeeper.registerDenom(token.Denom, banktypes.Metadata{
		Symbol: token.Symbol, Name: token.Name, Display: token.DisplayDenom, DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    token.Denom,
				Exponent: 0,
			},
			{
				Denom:    token.DisplayDenom,
				Exponent: uint32(token.Decimals),
			},
		},
	})
	bankKeeper.registerDenom(nativeDenom, banktypes.Metadata{
		Symbol: "NATIVE", Name: "Native Token", Display: evmtypes.GetEVMCoinDisplayDenom(), DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    nativeDenom,
				Exponent: 0,
			},
			{
				Denom:    evmtypes.GetEVMCoinDisplayDenom(),
				Exponent: 18,
			},
		},
	})
	bankKeeper.mint(mintTo.Bytes(), sdk.NewCoins(sdk.NewCoin(token.Denom, sdkmath.NewIntFromUint64(mintAmount))))
	bankKeeper.mint(mintTo.Bytes(), sdk.NewCoins(sdk.NewCoin(nativeDenom, sdkmath.NewIntFromUint64(mintAmount))))

	// map erc20
	erc20keeper.RegisterDenom(ctx, token.Denom, Erc20Address)

	return evm
}

func BenchmarkBankPrecompile(b *testing.B) {
	// remove MaxPrecompileCalls limit for benchmarks
	old := evmtypes.MaxPrecompileCalls
	evmtypes.MaxPrecompileCalls = 0
	defer func() {
		evmtypes.MaxPrecompileCalls = old
	}()

	token := TokenInfo{
		Denom:        "utoken",
		DisplayDenom: "token",
		Name:         "Test Token",
		Symbol:       "TTK",
		Decimals:     6,
	}
	mintAmount := uint64(1_000_000_000) // 1,000 TTK
	evm := Setup(b, token, MintTo, mintAmount)

	input, err := ABI.Pack("balances", MintTo)
	require.NoError(b, err)

	// Benchmark cases would go here
	b.ResetTimer()
	b.Run("Balances", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := evm.Call(MintTo, BankPrecompile, input, 1000000, common.U2560)
			require.NoError(b, err)
		}
	})
}
