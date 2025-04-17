package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	testconstants "github.com/cosmos/evm/testutil/constants"
	"github.com/cosmos/evm/x/precisebank/testutil"
	"github.com/cosmos/evm/x/precisebank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestGenesisStateValidate_Basic(t *testing.T) {
	configurator := evmtypes.NewEVMConfigurator()
	configurator.ResetTestConfig()
	err := configurator.WithEVMCoinInfo(testconstants.ExampleAttoDenom, uint8(evmtypes.SixDecimals)).Configure()
	require.NoError(t, err)

	testCases := []struct {
		name         string
		genesisState *types.GenesisState
		wantErr      string
	}{
		{
			"valid - default genesisState",
			types.DefaultGenesisState(),
			"",
		},
		{
			"valid - empty balances, zero remainder",
			types.NewGenesisState(
				types.FractionalBalances{},
				sdkmath.ZeroInt(),
				types.DefaultCoinInfo(),
			),
			"",
		},
		{
			"valid - nil balances",
			types.NewGenesisState(nil, sdkmath.ZeroInt(), types.DefaultCoinInfo()),
			"",
		},
		{
			"valid - max remainder amount",
			types.NewGenesisState(
				types.FractionalBalances{
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
				},
				types.ConversionFactor().SubRaw(1),
				types.DefaultCoinInfo(),
			),
			"",
		},
		{
			"invalid - empty genesisState (nil remainder)",
			&types.GenesisState{},
			"nil remainder amount",
		},
		{
			"valid - balances add up",
			types.NewGenesisState(
				types.FractionalBalances{
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
				},
				sdkmath.ZeroInt(),
				types.DefaultCoinInfo(),
			),
			"invalid balances: duplicate address cosmos1qyfkm2y3",
		},
		{
			"invalid - calls (single) FractionalBalance.Validate()",
			types.NewGenesisState(
				types.FractionalBalances{
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
					types.NewFractionalBalance(sdk.AccAddress{2}.String(), sdkmath.NewInt(-1)),
				},
				sdkmath.ZeroInt(),
				types.DefaultCoinInfo(),
			),
			"invalid balances: invalid fractional balance for cosmos1qgcgaq4k: non-positive amount -1",
		},
		{
			"invalid - calls (slice) FractionalBalances.Validate()",
			types.NewGenesisState(
				types.FractionalBalances{
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
				},
				sdkmath.ZeroInt(),
				types.DefaultCoinInfo(),
			),
			"invalid balances: duplicate address cosmos1qyfkm2y3",
		},
		{
			"invalid - negative remainder",
			types.NewGenesisState(
				types.FractionalBalances{
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
					types.NewFractionalBalance(sdk.AccAddress{2}.String(), sdkmath.NewInt(1)),
				},
				sdkmath.NewInt(-1),
				types.DefaultCoinInfo(),
			),
			"negative remainder amount -1",
		},
		{
			"invalid - too large remainder",
			types.NewGenesisState(
				types.FractionalBalances{
					types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.NewInt(1)),
					types.NewFractionalBalance(sdk.AccAddress{2}.String(), sdkmath.NewInt(1)),
				},
				types.ConversionFactor(),
				types.DefaultCoinInfo(),
			),
			"remainder 1000000000000 exceeds max of 999999999999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			err := tc.genesisState.Validate()

			if tc.wantErr == "" {
				require.NoError(tt, err)
			} else {
				require.Error(tt, err)
				require.EqualError(tt, err, tc.wantErr)
			}
		})
	}
}

func TestGenesisStateValidate_Total(t *testing.T) {
	testCases := []struct {
		name              string
		buildGenesisState func() *types.GenesisState
		containsErr       string
	}{
		{
			"valid - empty balances, zero remainder",
			func() *types.GenesisState {
				return types.NewGenesisState(nil, sdkmath.ZeroInt(), types.DefaultCoinInfo())
			},
			"",
		},
		{
			"valid - non-zero balances, zero remainder",
			func() *types.GenesisState {
				fbs := testutil.GenerateEqualFractionalBalances(t, 100)
				require.Len(t, fbs, 100)

				return types.NewGenesisState(fbs, sdkmath.ZeroInt(), types.DefaultCoinInfo())
			},
			"",
		},
		{
			"valid - non-zero balances, non-zero remainder",
			func() *types.GenesisState {
				fbs, remainder := testutil.GenerateEqualFractionalBalancesWithRemainder(t, 100)

				require.Len(t, fbs, 100)
				require.NotZero(t, remainder.Int64())

				t.Log("remainder:", remainder)

				return types.NewGenesisState(fbs, remainder, types.DefaultCoinInfo())
			},
			"",
		},
		{
			"invalid - non-zero balances, invalid remainder",
			func() *types.GenesisState {
				fbs, remainder := testutil.GenerateEqualFractionalBalancesWithRemainder(t, 100)

				require.Len(t, fbs, 100)
				require.NotZero(t, remainder.Int64())

				// Wrong remainder - should be non-zero
				return types.NewGenesisState(fbs, sdkmath.ZeroInt(), types.DefaultCoinInfo())
			},
			// balances are randomly generated so we can't set the exact value in the error message
			// "sum of fractional balances 52885778295370 ... "
			"+ remainder 0 is not a multiple of 1000000000000",
		},
		{
			"invalid - empty balances, non-zero remainder",
			func() *types.GenesisState {
				return types.NewGenesisState(types.FractionalBalances{}, sdkmath.NewInt(1), types.DefaultCoinInfo())
			},
			"sum of fractional balances 0 + remainder 1 is not a multiple of 1000000000000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(tt *testing.T) {
			err := tc.buildGenesisState().Validate()

			if tc.containsErr == "" {
				require.NoError(tt, err)
			} else {
				require.Error(tt, err)
				require.ErrorContains(tt, err, tc.containsErr)
			}
		})
	}
}

func TestGenesisState_TotalAmountWithRemainder(t *testing.T) {
	tests := []struct {
		name                         string
		giveBalances                 types.FractionalBalances
		giveRemainder                sdkmath.Int
		giveCoinInfo                 types.CoinInfo
		wantTotalAmountWithRemainder sdkmath.Int
	}{
		{
			"empty balances, zero remainder",
			types.FractionalBalances{},
			sdkmath.ZeroInt(),
			types.DefaultCoinInfo(),
			sdkmath.ZeroInt(),
		},
		{
			"non-empty balances, zero remainder",
			types.FractionalBalances{
				types.NewFractionalBalance(sdk.AccAddress{1}.String(), types.ConversionFactor().QuoRaw(2)),
				types.NewFractionalBalance(sdk.AccAddress{2}.String(), types.ConversionFactor().QuoRaw(2)),
			},
			sdkmath.ZeroInt(),
			types.DefaultCoinInfo(),
			types.ConversionFactor(),
		},
		{
			"non-empty balances, 1 remainder",
			types.FractionalBalances{
				types.NewFractionalBalance(sdk.AccAddress{1}.String(), types.ConversionFactor().QuoRaw(2)),
				types.NewFractionalBalance(sdk.AccAddress{2}.String(), types.ConversionFactor().QuoRaw(2).SubRaw(1)),
			},
			sdkmath.OneInt(),
			types.DefaultCoinInfo(),
			types.ConversionFactor(),
		},
		{
			"non-empty balances, max remainder",
			types.FractionalBalances{
				types.NewFractionalBalance(sdk.AccAddress{1}.String(), sdkmath.OneInt()),
			},
			types.ConversionFactor().SubRaw(1),
			types.DefaultCoinInfo(),
			types.ConversionFactor(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := types.NewGenesisState(
				tt.giveBalances,
				tt.giveRemainder,
				tt.giveCoinInfo,
			)

			require.NoError(t, gs.Validate(), "genesis state should be valid before testing total amount")

			totalAmt := gs.TotalAmountWithRemainder()
			require.Equal(t, tt.wantTotalAmountWithRemainder, totalAmt, "total amount should be balances + remainder")
		})
	}
}

func FuzzGenesisStateValidate_NonZeroRemainder(f *testing.F) {
	f.Add(5)
	f.Add(100)
	f.Add(30)

	f.Fuzz(func(t *testing.T, count int) {
		// Need at least 2 so we can generate both balances and remainder
		if count < 2 {
			t.Skip("count < 2")
		}

		fbs, remainder := testutil.GenerateEqualFractionalBalancesWithRemainder(t, count)

		t.Logf("count: %v", count)
		t.Logf("remainder: %v", remainder)

		gs := types.NewGenesisState(fbs, remainder, types.DefaultCoinInfo())
		require.NoError(t, gs.Validate())
	})
}

func FuzzGenesisStateValidate_ZeroRemainder(f *testing.F) {
	f.Add(5)
	f.Add(100)
	f.Add(30)

	f.Fuzz(func(t *testing.T, count int) {
		// Need at least 2 as 1 account with non-zero balance & no remainder is not valid
		if count < 2 {
			t.Skip("count < 2")
		}

		fbs := testutil.GenerateEqualFractionalBalances(t, count)

		gs := types.NewGenesisState(fbs, sdkmath.ZeroInt(), types.DefaultCoinInfo())
		require.NoError(t, gs.Validate())
	})
}
