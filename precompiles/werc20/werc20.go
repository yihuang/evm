package werc20

import (
	"embed"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ibcutils "github.com/cosmos/evm/ibc"
	cmn "github.com/cosmos/evm/precompiles/common"
	erc20 "github.com/cosmos/evm/precompiles/erc20"
	erc20types "github.com/cosmos/evm/x/erc20/types"
)

// abiPath defines the path to the WERC-20 precompile ABI JSON file.
const abiPath = "abi.json"

var (
	// Embed abi json file to the executable binary. Needed when importing as dependency.
	//
	//go:embed abi.json
	f   embed.FS
	ABI abi.ABI
)

func init() {
	var err error
	ABI, err = cmn.LoadABI(f, abiPath)
	if err != nil {
		panic(err)
	}
}

var (
	_ vm.PrecompiledContract = &Precompile{}
	_ cmn.NativeExecutor     = &Precompile{}
)

// Precompile defines the precompiled contract for WERC20.
type Precompile struct {
	*erc20.Precompile
}

const (
	// DepositRequiredGas defines the gas required for the Deposit transaction.
	DepositRequiredGas uint64 = 23_878
	// WithdrawRequiredGas defines the gas required for the Withdraw transaction.
	WithdrawRequiredGas uint64 = 9207
)

// NewPrecompile creates a new WERC20 Precompile instance implementing the
// PrecompiledContract interface. This type wraps around the ERC20 Precompile
// instance to provide additional methods.
func NewPrecompile(
	tokenPair erc20types.TokenPair,
	bankKeeper cmn.BankKeeper,
	erc20Keeper Erc20Keeper,
	transferKeeper ibcutils.TransferKeeper,
) (*Precompile, error) {
	erc20Precompile, err := erc20.NewPrecompile(tokenPair, bankKeeper, erc20Keeper, transferKeeper)
	if err != nil {
		return nil, fmt.Errorf("error instantiating the ERC20 precompile: %w", err)
	}

	// use the IWERC20 ABI
	erc20Precompile.ABI = ABI

	return &Precompile{
		Precompile: erc20Precompile,
	}, nil
}

// RequiredGas calculates the contract gas use.
func (p Precompile) RequiredGas(input []byte) uint64 {
	// TODO: these values were obtained from Remix using the WEVMOS9.sol.
	// We should execute the transactions from Cosmos EVM testnet
	// to ensure parity in the values.

	// If there is no method ID, then it's the fallback or receive case
	if len(input) < 4 {
		return DepositRequiredGas
	}

	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	case DepositMethod:
		return DepositRequiredGas
	case WithdrawMethod:
		return WithdrawRequiredGas
	default:
		return p.Precompile.RequiredGas(input)
	}
}

func (p Precompile) Execute(ctx sdk.Context, evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	method, args, err := cmn.SetupABI(p.ABI, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	stateDB := evm.StateDB

	switch {
	case method.Type == abi.Fallback,
		method.Type == abi.Receive,
		method.Name == DepositMethod:
		bz, err = p.Deposit(ctx, contract, stateDB)
	case method.Name == WithdrawMethod:
		bz, err = p.Withdraw(ctx, contract, stateDB, args)
	default:
		// ERC20 transactions and queries
		bz, err = p.HandleMethod(ctx, contract, stateDB, method, args)
	}

	return
}

// IsTransaction returns true if the given method name correspond to a
// transaction. Returns false otherwise.
func (p Precompile) IsTransaction(method *abi.Method) bool {
	txMethodName := []string{DepositMethod, WithdrawMethod}
	txMethodType := []abi.FunctionType{abi.Fallback, abi.Receive}

	if slices.Contains(txMethodName, method.Name) || slices.Contains(txMethodType, method.Type) {
		return true
	}

	return p.Precompile.IsTransaction(method)
}
