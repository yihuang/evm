package ics20

import (
	"embed"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	storetypes "cosmossdk.io/store/types"
)

// PrecompileAddress of the ICS-20 EVM extension in hex format.
const PrecompileAddress = "0x0000000000000000000000000000000000000802"

var (
	_ vm.PrecompiledContract = &Precompile{}
	_ cmn.NativeExecutor     = &Precompile{}
)

var (
	// Embed abi json file to the executable binary. Needed when importing as dependency.
	//
	//go:embed abi.json
	f   embed.FS
	ABI abi.ABI
)

func init() {
	var err error
	ABI, err = cmn.LoadABI(f, "abi.json")
	if err != nil {
		panic(err)
	}
}

type Precompile struct {
	cmn.Precompile

	abi.ABI
	bankKeeper     cmn.BankKeeper
	stakingKeeper  cmn.StakingKeeper
	transferKeeper cmn.TransferKeeper
	channelKeeper  cmn.ChannelKeeper
}

// NewPrecompile creates a new ICS-20 Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	bankKeeper cmn.BankKeeper,
	stakingKeeper cmn.StakingKeeper,
	transferKeeper cmn.TransferKeeper,
	channelKeeper cmn.ChannelKeeper,
) (*Precompile, error) {
	p := &Precompile{
		Precompile: cmn.Precompile{
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ContractAddress:      common.HexToAddress(evmtypes.ICS20PrecompileAddress),
			BalanceHandler:       cmn.NewBalanceHandler(bankKeeper),
		},
		ABI:            ABI,
		bankKeeper:     bankKeeper,
		transferKeeper: transferKeeper,
		channelKeeper:  channelKeeper,
		stakingKeeper:  stakingKeeper,
	}
	p.Executor = p
	return p, nil
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}

	return p.Precompile.RequiredGas(input, p.IsTransaction(method))
}

func (p Precompile) Execute(ctx sdk.Context, evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
	method, args, err := cmn.SetupABI(p.ABI, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB
	var bz []byte

	switch method.Name {
	// ICS20 transactions
	case TransferMethod:
		bz, err = p.Transfer(ctx, contract, stateDB, method, args)
	// ICS20 queries
	case DenomMethod:
		bz, err = p.Denom(ctx, contract, method, args)
	case DenomsMethod:
		bz, err = p.Denoms(ctx, contract, method, args)
	case DenomHashMethod:
		bz, err = p.DenomHash(ctx, contract, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	return bz, err
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
//
// Available ics20 transactions are:
//   - Transfer
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case TransferMethod:
		return true
	default:
		return false
	}
}
