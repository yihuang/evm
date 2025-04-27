package types_test

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	testconstants "github.com/cosmos/evm/testutil/constants"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestConvertCoinsFrom18Decimals(t *testing.T) {
	eighteenDecimalsCoinInfo := testconstants.ExampleChainCoinInfo[testconstants.ExampleChainID]
	sixDecimalsCoinInfo := testconstants.ExampleChainCoinInfo[testconstants.SixDecimalsChainID]

	nonBaseCoin := sdk.Coin{Denom: "btc", Amount: math.NewInt(10)}
	eighteenDecimalsBaseCoin := sdk.Coin{Denom: eighteenDecimalsCoinInfo.Denom, Amount: math.NewInt(10)}
	sixDecimalsBaseCoin := sdk.Coin{Denom: sixDecimalsCoinInfo.Denom, Amount: math.NewInt(10)}

	testCases := []struct {
		name        string
		evmCoinInfo evmtypes.EvmCoinInfo
		coins       sdk.Coins
		expCoins    sdk.Coins
	}{
		{
			name:        "pass - no evm denom",
			evmCoinInfo: sixDecimalsCoinInfo,
			coins:       sdk.Coins{nonBaseCoin},
			expCoins:    sdk.Coins{nonBaseCoin},
		},
		{
			name:        "pass - only base denom 18 decimals",
			evmCoinInfo: eighteenDecimalsCoinInfo,
			coins:       sdk.Coins{eighteenDecimalsBaseCoin},
			expCoins:    sdk.Coins{eighteenDecimalsBaseCoin},
		},
		{
			name:        "pass - only base denom 6 decimals",
			evmCoinInfo: sixDecimalsCoinInfo,
			coins:       sdk.Coins{sixDecimalsBaseCoin},
			expCoins:    sdk.Coins{sdk.Coin{Denom: sixDecimalsCoinInfo.ExtendedDenom, Amount: math.NewInt(10)}},
		},
		{
			name:        "pass - multiple coins and base denom 18 decimals",
			evmCoinInfo: eighteenDecimalsCoinInfo,
			coins:       sdk.Coins{nonBaseCoin, eighteenDecimalsBaseCoin}.Sort(),
			expCoins:    sdk.Coins{nonBaseCoin, eighteenDecimalsBaseCoin}.Sort(),
		},
		{
			name:        "pass - multiple coins and base denom 6 decimals",
			evmCoinInfo: sixDecimalsCoinInfo,
			coins:       sdk.Coins{nonBaseCoin, sixDecimalsBaseCoin}.Sort(),
			expCoins:    sdk.Coins{nonBaseCoin, sdk.Coin{Denom: sixDecimalsCoinInfo.ExtendedDenom, Amount: math.NewInt(10)}}.Sort(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configurator := evmtypes.NewEVMConfigurator()
			configurator.ResetTestConfig()
			require.NoError(t, configurator.WithEVMCoinInfo(tc.evmCoinInfo).Configure())

			coinConverted := evmtypes.ConvertCoinsFrom18Decimals(tc.coins)
			require.Equal(t, tc.expCoins, coinConverted, "expected a different coin")
		})
	}
}

func TestConvertBigIntFrom18DecimalsToLegacyDec(t *testing.T) {
	testCases := []struct {
		name    string
		amt     *big.Int
		exp6dec math.LegacyDec
	}{
		{
			name:    "smallest amount",
			amt:     big.NewInt(1),
			exp6dec: math.LegacyMustNewDecFromStr("0.000000000001"),
		},
		{
			name:    "almost 1: 0.99999...",
			amt:     big.NewInt(999999999999),
			exp6dec: math.LegacyMustNewDecFromStr("0.999999999999"),
		},
		{
			name:    "half of the minimum uint",
			amt:     big.NewInt(5e11),
			exp6dec: math.LegacyMustNewDecFromStr("0.5"),
		},
		{
			name:    "one int",
			amt:     big.NewInt(1e12),
			exp6dec: math.LegacyOneDec(),
		},
		{
			name:    "one 'ether'",
			amt:     big.NewInt(1e18),
			exp6dec: math.LegacyNewDec(1e6),
		},
	}

	for _, coinInfo := range []evmtypes.EvmCoinInfo{
		testconstants.ExampleChainCoinInfo[testconstants.SixDecimalsChainID],
		testconstants.ExampleChainCoinInfo[testconstants.ExampleChainID],
	} {
		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%d dec - %s", coinInfo.Decimals, tc.name), func(t *testing.T) {
				configurator := evmtypes.NewEVMConfigurator()
				configurator.ResetTestConfig()
				require.NoError(t, configurator.WithEVMCoinInfo(coinInfo).Configure())
				res := evmtypes.ConvertBigIntFrom18DecimalsToLegacyDec(tc.amt)
				exp := math.LegacyNewDecFromBigInt(tc.amt)
				if coinInfo.Decimals == evmtypes.SixDecimals {
					exp = tc.exp6dec
				}
				require.Equal(t, exp, res)
			})
		}
	}
}
