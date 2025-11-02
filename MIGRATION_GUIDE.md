# Precompile Test Migration Guide to go-abi

## Overview

This guide explains how to migrate precompile integration tests from the old go-ethereum ABI API to the new custom `go-abi` library (github.com/yihuang/go-abi).

## Background

The precompiles have been refactored to use a custom `go-abi` library instead of the standard `go-ethereum/accounts/abi` package. This change requires updates to the test code to work with the new API.

## Key Differences

### Old API (go-ethereum/accounts/abi)
- Used `abi.ABI.Pack()` to encode method calls
- Used `UnpackIntoInterface()` and `Unpack()` to decode results
- Relied on reflection for encoding/decoding
- Tests accessed `precompile.ABI` field

### New API (go-abi)
- Uses generated types with methods like `EncodeWithSelector()`
- Uses generated `Decode()` methods on result types
- No reflection, fully type-safe
- No `ABI` field on precompile struct

## Migration Steps

### 1. Update ContractData Struct

**Before:**
```go
type ContractData struct {
    ownerPriv cryptotypes.PrivKey
    contractAddr   common.Address
    contractABI    abi.ABI
    precompileAddr common.Address
    precompileABI  abi.ABI  // ← Remove this
}
```

**After:**
```go
type ContractData struct {
    ownerPriv cryptotypes.PrivKey
    contractAddr   common.Address
    contractABI    abi.ABI
    precompileAddr common.Address
    // precompileABI removed
}
```

### 2. Remove precompileABI from initialization

**Before:**
```go
contractData = ContractData{
    ownerPriv:      sender.Priv,
    precompileAddr: is.precompile.Address(),
    precompileABI:  is.precompile.ABI,  // ← Remove this
    contractAddr:   bankCallerContractAddr,
    contractABI:    bankCallerContract.ABI,
}
```

**After:**
```go
contractData = ContractData{
    ownerPriv:      sender.Priv,
    precompileAddr: is.precompile.Address(),
    contractAddr:   bankCallerContractAddr,
    contractABI:    bankCallerContract.ABI,
}
```

### 3. Create Helper Functions

Add these helper functions to your test_utils.go:

```go
// decodeBalancesResult decodes the result from a balances query
func decodeBalancesResult(data []byte) ([]bank.Balance, error) {
    var result bank.BalancesReturn
    _, err := result.Decode(data)
    if err != nil {
        return nil, err
    }
    return result.Balances, nil
}

// decodeTotalSupplyResult decodes the result from a totalSupply query
func decodeTotalSupplyResult(data []byte) ([]bank.Balance, error) {
    var result bank.TotalSupplyReturn
    _, err := result.Decode(data)
    if err != nil {
        return nil, err
    }
    return result.TotalSupply, nil
}

// decodeSupplyOfResult decodes the result from a supplyOf query
func decodeSupplyOfResult(data []byte) (*big.Int, error) {
    var result bank.SupplyOfReturn
    _, err := result.Decode(data)
    if err != nil {
        return nil, err
    }
    return result.TotalSupply, nil
}
```

### 4. Update getTxAndCallArgs Function

This function handles encoding for direct precompile calls. Replace manual encoding with `EncodeWithSelector()`:

```go
func getTxAndCallArgs(
    callType int,
    contractData ContractData,
    methodName string,
    args ...interface{},
) (evmtypes.EvmTxArgs, testutiltypes.CallArgs) {
    txArgs := evmtypes.EvmTxArgs{}
    callArgs := testutiltypes.CallArgs{}

    switch callType {
    case directCall:
        var input []byte
        switch methodName {
        case bank.BalancesMethod:
            addr := args[0].(common.Address)
            call := bank.BalancesCall{Account: addr}
            input, _ = call.EncodeWithSelector()  // Use built-in method
        case bank.TotalSupplyMethod:
            var call bank.TotalSupplyCall
            input, _ = call.EncodeWithSelector()
        case bank.SupplyOfMethod:
            addr := args[0].(common.Address)
            call := bank.SupplyOfCall{Erc20Address: addr}
            input, _ = call.EncodeWithSelector()
        default:
            panic(fmt.Sprintf("unknown method: %s", methodName))
        }
        txArgs.To = &contractData.precompileAddr
        txArgs.Input = input
        callArgs.ContractABI = abi.ABI{}
    case contractCall:
        txArgs.To = &contractData.contractAddr
        callArgs.ContractABI = contractData.contractABI
    }

    callArgs.MethodName = methodName
    callArgs.Args = args
    return txArgs, callArgs
}
```

### 5. Replace UnpackIntoInterface Calls

**Before:**
```go
var balances []bank.Balance
err = is.precompile.UnpackIntoInterface(&balances, bank2.BalancesMethod, ethRes.Ret)
Expect(err).ToNot(HaveOccurred(), "failed to unpack balances")
```

**After:**
```go
balances, err := decodeBalancesResult(ethRes.Ret)
Expect(err).ToNot(HaveOccurred(), "failed to unpack balances")
```

### 6. Replace Unpack Calls

**Before:**
```go
out, err := is.precompile.Unpack(bank2.SupplyOfMethod, ethRes.Ret)
Expect(err).ToNot(HaveOccurred(), "failed to unpack balances")
Expect(out[0].(*big.Int).String()).To(Equal(expectedValue.String()))
```

**After:**
```go
supply, err := decodeSupplyOfResult(ethRes.Ret)
Expect(err).ToNot(HaveOccurred(), "failed to unpack balances")
Expect(supply.String()).To(Equal(expectedValue.String()))
```

### 7. Update Unit Tests (test_query.go)

**Before:**
```go
func (s *PrecompileTestSuite) TestBalances() {
    s.SetupTest()
    method := s.precompile.Methods[bank.BalancesMethod]  // ← Methods field doesn't exist

    // Test cases using method variable...
    bz, err := s.precompile.Balances(ctx, &method, args)
    var balances []bank.Balance
    err = s.precompile.UnpackIntoInterface(&balances, method.Name, bz)
}
```

**After:**
```go
func (s *PrecompileTestSuite) TestBalances() {
    s.SetupTest()

    // Test cases use typed call structs directly
    call := &bank.BalancesCall{Account: addr}
    result, err := s.precompile.Balances(ctx, call)

    balances := result.Balances  // Direct access to result fields
}
```

## Generated Types

The `go-abi` tool generates these types for each method:

- `{MethodName}Call` - Input parameters struct
- `{MethodName}Return` - Output results struct
- `{MethodName}Selector` - Method selector constant
- `{MethodName}ID` - Method ID constant
- `{MethodName}Method` - Method name constant

## Example for Other Precompiles

For each precompile (bech32, distribution, erc20, gov, etc.), you need to:

1. Check the generated types in `{precompile}.abi.go`
2. Create appropriate decode helper functions
3. Update `getTxAndCallArgs` to handle the precompile's methods
4. Replace all `UnpackIntoInterface` and `Unpack` calls
5. Update unit tests to use typed call structs

## Verification

After migration, verify the tests build successfully:

```bash
go build -tags=tests ./tests/integration/precompiles/{precompile_name}/...
```

## References

- Bank precompile migration: `tests/integration/precompiles/bank/`
- go-abi library: github.com/yihuang/go-abi
- Generated types example: `precompiles/bank/bank.abi.go`
