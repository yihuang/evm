package types

import "math/big"

// ConvertAmountTo18DecimalsBigInt convert the given amount into a 18 decimals
// representation.
//
// NOTE: With the introduction of the precisebank module, all scaling-related code
// has been removed except for the ConvertAmountTo18DecimalsBigInt function.
// The reason this function remains is that we need to convert the amount
// from the precompile of staking, distribution, erc20, etc. to 18 decimals
// to add SetBalanceChangeEntries.
func ConvertAmountTo18DecimalsBigInt(amt *big.Int) *big.Int {
	evmCoinDecimal := GetEVMCoinDecimals()

	return new(big.Int).Mul(amt, evmCoinDecimal.ConversionFactor().BigInt())
}
