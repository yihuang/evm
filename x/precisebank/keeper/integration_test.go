package keeper_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/cosmos/evm/contracts"
	testconstants "github.com/cosmos/evm/testutil/constants"
	"github.com/cosmos/evm/testutil/integration/os/factory"
	"github.com/cosmos/evm/testutil/integration/os/utils"
	"github.com/cosmos/evm/x/precisebank/keeper"
	"github.com/cosmos/evm/x/precisebank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (suite *KeeperIntegrationTestSuite) TestMintBurnSendCoins_RandomValueMultiDecimals() {
	tests := []struct {
		name    string
		chainId string
	}{
		{
			name:    "6 decimals",
			chainId: testconstants.SixDecimalsChainID,
		},
		{
			name:    "2 decimals",
			chainId: testconstants.TwoDecimalsChainID,
		},
		{
			name:    "12 decimals",
			chainId: testconstants.TwelveDecimalsChainID,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()
			ctx := suite.network.GetContext()

			ethCfg := evmtypes.DefaultChainConfig(tt.chainId)
			coinInfo := testconstants.ExampleChainCoinInfo[tt.chainId]

			configurator := evmtypes.NewEVMConfigurator()
			configurator.ResetTestConfig()
			configurator.
				WithChainConfig(ethCfg).
				WithEVMCoinInfo(coinInfo)
			err := configurator.Configure()
			suite.Require().NoError(err)

			moduleName := evmtypes.ModuleName
			sender := sdk.AccAddress([]byte{1})
			recipient := sdk.AccAddress([]byte{2})

			// Mint initial balance to sender
			initialBalance := types.ConversionFactor().MulRaw(100)
			initialCoins := cs(ci(types.ExtendedCoinDenom, initialBalance))
			suite.Require().NoError(suite.network.App.PreciseBankKeeper.MintCoins(ctx, moduleName, initialCoins))
			suite.Require().NoError(suite.network.App.PreciseBankKeeper.SendCoinsFromModuleToAccount(ctx, moduleName, sender, initialCoins))

			maxUnit := types.ConversionFactor().MulRaw(2).SubRaw(1)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))

			// Expected balances tracking
			expectedSenderBal := initialBalance
			expectedRecipientBal := sdkmath.ZeroInt()

			mintCount, burnCount, sendCount := 0, 0, 0

			mintAmount := sdkmath.NewInt(0)
			burnAmount := sdkmath.NewInt(0)

			iterations := 1000
			for range iterations {
				op := r.Intn(3)
				switch op {
				case 0: // Mint to sender via module
					randAmount := sdkmath.NewIntFromBigInt(new(big.Int).Rand(r, maxUnit.BigInt())).AddRaw(1)
					mintCoins := cs(ci(types.ExtendedCoinDenom, randAmount))
					if err := suite.network.App.PreciseBankKeeper.MintCoins(ctx, moduleName, mintCoins); err != nil {
						continue
					}
					if err := suite.network.App.PreciseBankKeeper.SendCoinsFromModuleToAccount(ctx, moduleName, sender, mintCoins); err != nil {
						continue
					}
					expectedSenderBal = expectedSenderBal.Add(randAmount)
					mintAmount = mintAmount.Add(randAmount)
					mintCount++

				case 1: // Burn from sender via module
					senderBal := suite.GetAllBalances(sender).AmountOf(types.ExtendedCoinDenom)
					if senderBal.IsZero() {
						continue
					}
					burnable := sdkmath.MinInt(senderBal, maxUnit)
					randAmount := sdkmath.NewIntFromBigInt(new(big.Int).Rand(r, burnable.BigInt())).AddRaw(1)
					burnCoins := cs(ci(types.ExtendedCoinDenom, randAmount))
					if err := suite.network.App.PreciseBankKeeper.SendCoinsFromAccountToModule(ctx, sender, moduleName, burnCoins); err != nil {
						continue
					}
					if err := suite.network.App.PreciseBankKeeper.BurnCoins(ctx, moduleName, burnCoins); err != nil {
						continue
					}
					expectedSenderBal = expectedSenderBal.Sub(randAmount)
					burnAmount = burnAmount.Add(randAmount)
					burnCount++

				case 2: // Send from sender to recipient
					senderBal := suite.GetAllBalances(sender).AmountOf(types.ExtendedCoinDenom)
					if senderBal.IsZero() {
						continue
					}
					sendable := sdkmath.MinInt(senderBal, maxUnit)
					randAmount := sdkmath.NewIntFromBigInt(new(big.Int).Rand(r, sendable.BigInt())).AddRaw(1)
					sendCoins := cs(ci(types.ExtendedCoinDenom, randAmount))
					if err := suite.network.App.PreciseBankKeeper.SendCoins(ctx, sender, recipient, sendCoins); err != nil {
						continue
					}
					expectedSenderBal = expectedSenderBal.Sub(randAmount)
					expectedRecipientBal = expectedRecipientBal.Add(randAmount)
					sendCount++
				}
			}

			suite.T().Logf("Executed operations: %d mints, %d burns, %d sends", mintCount, burnCount, sendCount)

			// Check balances
			actualSenderBal := suite.GetAllBalances(sender).AmountOf(types.ExtendedCoinDenom)
			actualRecipientBal := suite.GetAllBalances(recipient).AmountOf(types.ExtendedCoinDenom)
			suite.Require().Equal(expectedSenderBal.BigInt().Cmp(actualSenderBal.BigInt()), 0, "Sender balance mismatch (expected: %s, actual: %s)", expectedSenderBal, actualSenderBal)
			suite.Require().Equal(expectedRecipientBal.BigInt().Cmp(actualRecipientBal.BigInt()), 0, "Recipient balance mismatch (expected: %s, actual: %s)", expectedRecipientBal, actualRecipientBal)

			// Check remainder
			expectedRemainder := burnAmount.Sub(mintAmount).Mod(types.ConversionFactor())
			actualRemainder := suite.network.App.PreciseBankKeeper.GetRemainderAmount(ctx)
			suite.Require().Equal(expectedRemainder.BigInt().Cmp(actualRemainder.BigInt()), 0, "Remainder mismatch (expected: %s, actual: %s)", expectedRemainder, actualRemainder)

			// Invariant check
			inv := keeper.AllInvariants(suite.network.App.PreciseBankKeeper)
			res, stop := inv(ctx)
			suite.Require().False(stop, "Invariant broken")
			suite.Require().Empty(res, "Unexpected invariant violation: %s", res)
		})
	}
}

func (suite *KeeperIntegrationTestSuite) TestWATOMWrapUnwrap_MultiDecimal() {
	tests := []struct {
		name    string
		chainId string
	}{
		{
			name:    "6 decimals",
			chainId: testconstants.SixDecimalsChainID,
		},
		{
			name:    "12 decimals",
			chainId: testconstants.TwelveDecimalsChainID,
		},
		{
			name:    "2 decimals",
			chainId: testconstants.TwoDecimalsChainID,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTestWithChainID(tt.chainId)
			ctx := suite.network.GetContext()

			ethCfg := evmtypes.DefaultChainConfig(tt.chainId)
			coinInfo := testconstants.ExampleChainCoinInfo[tt.chainId]

			configurator := evmtypes.NewEVMConfigurator()
			configurator.ResetTestConfig()
			configurator.
				WithChainConfig(ethCfg).
				WithEVMCoinInfo(coinInfo)
			err := configurator.Configure()
			suite.Require().NoError(err)

			sender := suite.keyring.GetKey(0)
			amount := big.NewInt(1)

			// Deploy WATOM contract
			watomAddr, err := suite.factory.DeployContract(
				sender.Priv,
				evmtypes.EvmTxArgs{},
				factory.ContractDeploymentData{
					Contract: contracts.WATOMContract,
				},
			)
			suite.Require().NoError(err)

			err = suite.network.NextBlock()
			suite.Require().NoError(err)

			baseFeeRes, err := suite.network.GetEvmClient().BaseFee(ctx, &evmtypes.QueryBaseFeeRequest{})
			suite.Require().NoError(err)

			// Call deposit() with msg.value = wrapAmount
			_, err = suite.factory.ExecuteContractCall(
				sender.Priv,
				evmtypes.EvmTxArgs{
					To:        &watomAddr,
					Amount:    amount,
					GasLimit:  100_000,
					GasFeeCap: baseFeeRes.BaseFee.BigInt(),
					GasTipCap: big.NewInt(1),
				},
				factory.CallArgs{
					ContractABI: contracts.WATOMContract.ABI,
					MethodName:  "deposit",
				},
			)
			suite.Require().NoError(err)
			err = suite.network.NextBlock()
			suite.Require().NoError(err)

			// Check WATOM balance == wrapAmount
			bal, err := utils.GetERC20Balance(suite.network, watomAddr, sender.Addr)
			suite.Require().NoError(err)
			suite.Require().Equal(amount.Cmp(bal), 0, "WATOM balance should match deposited amount (expected: %s, actual: %s)", amount, bal)

			baseFeeRes, err = suite.network.GetEvmClient().BaseFee(ctx, &evmtypes.QueryBaseFeeRequest{})
			suite.Require().NoError(err)

			fmt.Println(suite.network.App.PreciseBankKeeper.GetAllBalances(ctx, watomAddr.Bytes()))

			// Call withdraw(wrapAmount)
			_, err = suite.factory.ExecuteContractCall(
				sender.Priv,
				evmtypes.EvmTxArgs{
					To:        &watomAddr,
					GasLimit:  100_000,
					GasFeeCap: baseFeeRes.BaseFee.BigInt(),
					GasTipCap: big.NewInt(1),
				},
				factory.CallArgs{
					ContractABI: contracts.WATOMContract.ABI,
					MethodName:  "withdraw",
					Args:        []interface{}{amount},
				},
			)
			suite.Require().NoError(err)
			suite.Require().NoError(suite.network.NextBlock())

			// Final WATOM balance should be 0
			bal, err = utils.GetERC20Balance(suite.network, watomAddr, sender.Addr)
			suite.Require().NoError(err)
			suite.Require().Equal("0", bal.String(), "WATOM balance should be zero after withdraw")
		})
	}
}
