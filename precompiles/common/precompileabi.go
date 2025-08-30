package common

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

// SetupABI runs the initial setup required to run a transaction or a query.
// It returns the ABI method, initial gas and calling arguments.
func SetupABI(
	api abi.ABI,
	contract *vm.Contract,
	readOnly bool,
	isTransaction func(name *abi.Method) bool,
) (method *abi.Method, args []interface{}, err error) {
	// NOTE: This is a special case where the calling transaction does not specify a function name.
	// In this case we default to a `fallback` or `receive` function on the contract.

	// Simplify the calldata checks
	isEmptyCallData := len(contract.Input) == 0
	isShortCallData := len(contract.Input) > 0 && len(contract.Input) < 4
	isStandardCallData := len(contract.Input) >= 4

	switch {
	// Case 1: Calldata is empty
	case isEmptyCallData:
		method, err = emptyCallData(api, contract)

	// Case 2: calldata is non-empty but less than 4 bytes needed for a method
	case isShortCallData:
		method, err = methodIDCallData(api)

	// Case 3: calldata is non-empty and contains the minimum 4 bytes needed for a method
	case isStandardCallData:
		method, err = standardCallData(api, contract)
	}

	if err != nil {
		return nil, nil, err
	}

	// return error if trying to write to state during a read-only call
	if readOnly && isTransaction(method) {
		return nil, nil, vm.ErrWriteProtection
	}

	// if the method type is `function` continue looking for arguments
	if method.Type == abi.Function {
		argsBz := contract.Input[4:]
		args, err = method.Inputs.Unpack(argsBz)
		if err != nil {
			return nil, nil, err
		}
	}

	return method, args, nil
}

// emptyCallData is a helper function that returns the method to be called when the calldata is empty.
func emptyCallData(api abi.ABI, contract *vm.Contract) (method *abi.Method, err error) {
	switch {
	// Case 1.1: Send call or transfer tx - 'receive' is called if present and value is transferred
	case contract.Value().Sign() > 0 && api.HasReceive():
		return &api.Receive, nil
	// Case 1.2: Either 'receive' is not present, or no value is transferred - call 'fallback' if present
	case api.HasFallback():
		return &api.Fallback, nil
	// Case 1.3: Neither 'receive' nor 'fallback' are present - return error
	default:
		return nil, vm.ErrExecutionReverted
	}
}

// methodIDCallData is a helper function that returns the method to be called when the calldata is less than 4 bytes.
func methodIDCallData(api abi.ABI) (method *abi.Method, err error) {
	// Case 2.2: calldata contains less than 4 bytes needed for a method and 'fallback' is not present - return error
	if !api.HasFallback() {
		return nil, vm.ErrExecutionReverted
	}
	// Case 2.1: calldata contains less than 4 bytes needed for a method - 'fallback' is called if present
	return &api.Fallback, nil
}

// standardCallData is a helper function that returns the method to be called when the calldata is 4 bytes or more.
func standardCallData(api abi.ABI, contract *vm.Contract) (method *abi.Method, err error) {
	methodID := contract.Input[:4]
	// NOTE: this function iterates over the method map and returns
	// the method with the given ID
	method, err = api.MethodById(methodID)

	// Case 3.1 calldata contains a non-existing method ID, and `fallback` is not present - return error
	if err != nil && !api.HasFallback() {
		return nil, err
	}

	// Case 3.2: calldata contains a non-existing method ID - 'fallback' is called if present
	if err != nil && api.HasFallback() {
		return &api.Fallback, nil
	}

	return method, nil
}
