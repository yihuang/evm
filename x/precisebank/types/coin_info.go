package types

import (
	"fmt"

	testconstants "github.com/cosmos/evm/testutil/constants"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewCoinInfo(
	integerCoinDenom string,
	extendedCoinDenom string,
	conversionFactor sdkmath.Int,
) CoinInfo {
	return CoinInfo{
		IntegerCoinDenom:  integerCoinDenom,
		ExtendedCoinDenom: extendedCoinDenom,
		ConversionFactor:  conversionFactor,
	}
}

// default precisebank module parameter
// 6 decimals uatom <-> 18 decimals aatom
func DefaultCoinInfo() CoinInfo {
	return CoinInfo{
		IntegerCoinDenom:  testconstants.ExampleMicroDenom,
		ExtendedCoinDenom: testconstants.ExampleAttoDenom,
		ConversionFactor:  evmtypes.SixDecimals.ConversionFactor(),
	}
}

func (p CoinInfo) Validate() error {
	if sdk.ValidateDenom(p.IntegerCoinDenom) != nil {
		return fmt.Errorf("invalid integer coin denom: %s", p.IntegerCoinDenom)
	}
	if sdk.ValidateDenom(p.ExtendedCoinDenom) != nil {
		return fmt.Errorf("invalid extended coin denom: %s", p.ExtendedCoinDenom)
	}
	// check if 1 <= conversion factor <= 1e17
	if p.ConversionFactor.LT(sdkmath.OneInt()) || p.ConversionFactor.GT(sdkmath.NewInt(1e17)) {
		return fmt.Errorf("invalid conversion factor: %s", p.ConversionFactor.String())
	}
	// check if conversion factor is a power of 10
	temp := p.ConversionFactor
	for !temp.Equal(sdkmath.OneInt()) {
		if !temp.ModRaw(10).IsZero() {
			return fmt.Errorf("conversion factor must be a power of 10: %s", p.ConversionFactor)
		}
		temp = temp.QuoRaw(10)
	}

	return nil
}
