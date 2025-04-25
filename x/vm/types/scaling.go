package types

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ConvertAmountToLegacy18Decimals convert the given amount into a 18 decimals
// representation.
func ConvertAmountTo18DecimalsLegacy(amt sdkmath.LegacyDec) sdkmath.LegacyDec {
	evmCoinDecimal := GetEVMCoinDecimals()

	return amt.MulInt(evmCoinDecimal.ConversionFactor())
}

// ConvertAmountTo18DecimalsBigInt convert the given amount into a 18 decimals
// representation.
func ConvertAmountTo18DecimalsBigInt(amt *big.Int) *big.Int {
	evmCoinDecimal := GetEVMCoinDecimals()

	return new(big.Int).Mul(amt, evmCoinDecimal.ConversionFactor().BigInt())
}

// ConvertAmountFrom18DecimalsBigInt convert the given amount into a 18 decimals
// representation.
func ConvertAmountFrom18DecimalsBigInt(amt *big.Int) *big.Int {
	evmCoinDecimal := GetEVMCoinDecimals()

	return new(big.Int).Quo(amt, evmCoinDecimal.ConversionFactor().BigInt())
}

// ConvertBigIntFrom18DecimalsToLegacyDec converts the given amount into a LegacyDec
// with the corresponding decimals of the EVM denom.
func ConvertBigIntFrom18DecimalsToLegacyDec(amt *big.Int) sdkmath.LegacyDec {
	evmCoinDecimal := GetEVMCoinDecimals()
	decAmt := sdkmath.LegacyNewDecFromBigInt(amt)
	return decAmt.QuoInt(evmCoinDecimal.ConversionFactor())
}

// ConvertEvmCoinFrom18Decimals converts the coin's Amount from 18 decimals to its
// original representation. Return an error if the coin denom is not the EVM.
func ConvertEvmCoinFrom18Decimals(coin sdk.Coin) (sdk.Coins, error) {
	if coin.Denom != GetEVMCoinDenom() {
		return sdk.Coins{}, fmt.Errorf("expected coin denom %s, received %s", GetEVMCoinDenom(), coin.Denom)
	}

	if GetEVMCoinDecimals() == EighteenDecimals {
		return sdk.NewCoins(coin), nil
	}

	evmCoinDecimal := GetEVMCoinDecimals()
	integerAmount := coin.Amount.Quo(evmCoinDecimal.ConversionFactor())
	fractionalAmount := coin.Amount.Mod(evmCoinDecimal.ConversionFactor())

	integerCoin := sdk.NewCoin(GetEVMCoinDenom(), integerAmount)
	fractionalCoin := sdk.NewCoin(GetEVMCoinExtendedDenom(), fractionalAmount)
	return sdk.NewCoins(integerCoin, fractionalCoin), nil
}

// ConvertCoinsFrom18Decimals returns the given coins with the Amount of the evm
// coin converted from the 18 decimals representation to the original one.
func ConvertCoinsFrom18Decimals(coins sdk.Coins) sdk.Coins {
	evmCoinDecimal := GetEVMCoinDecimals()
	if evmCoinDecimal == EighteenDecimals {
		return coins
	}

	evmDenom := GetEVMCoinDenom()
	evmExtendedDenom := GetEVMCoinExtendedDenom()

	var newCoins sdk.Coins
	for _, coin := range coins {
		if coin.Denom == evmDenom {
			conversionFactor := evmCoinDecimal.ConversionFactor()
			integerAmt := coin.Amount.Quo(conversionFactor)
			fractionalAmt := coin.Amount.Mod(conversionFactor)

			integerCoin := sdk.NewCoin(evmDenom, integerAmt)
			fractionalCoin := sdk.NewCoin(evmExtendedDenom, fractionalAmt)
			convertedCoins := sdk.NewCoins(integerCoin, fractionalCoin)
			newCoins = newCoins.Add(convertedCoins...)
		} else {
			newCoins = newCoins.Add(coin)
		}
	}
	return newCoins
}
