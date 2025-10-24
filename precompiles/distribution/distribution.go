package distribution

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	cmn "github.com/cosmos/evm/precompiles/common"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	_ "embed"
)

//go:generate go run github.com/yihuang/go-abi/cmd -input abi.json -output distribution.abi.go -external-tuples Coin=cmn.Coin,Dec=cmn.Dec,DecCoin=cmn.DecCoin -imports cmn=github.com/cosmos/evm/precompiles/common

var _ vm.PrecompiledContract = &Precompile{}

// Precompile defines the precompiled contract for distribution.
type Precompile struct {
	cmn.Precompile

	distributionKeeper    cmn.DistributionKeeper
	distributionMsgServer distributiontypes.MsgServer
	distributionQuerier   distributiontypes.QueryServer
	stakingKeeper         cmn.StakingKeeper
	addrCdc               address.Codec
}

// NewPrecompile creates a new distribution Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	distributionKeeper cmn.DistributionKeeper,
	distributionMsgServer distributiontypes.MsgServer,
	distributionQuerier distributiontypes.QueryServer,
	stakingKeeper cmn.StakingKeeper,
	bankKeeper cmn.BankKeeper,
	addrCdc address.Codec,
) *Precompile {
	return &Precompile{
		Precompile: cmn.Precompile{
			KvGasConfig:           storetypes.KVGasConfig(),
			TransientKVGasConfig:  storetypes.TransientGasConfig(),
			ContractAddress:       common.HexToAddress(evmtypes.DistributionPrecompileAddress),
			BalanceHandlerFactory: cmn.NewBalanceHandlerFactory(bankKeeper),
		},
		stakingKeeper:         stakingKeeper,
		distributionKeeper:    distributionKeeper,
		distributionMsgServer: distributionMsgServer,
		distributionQuerier:   distributionQuerier,
		addrCdc:               addrCdc,
	}
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	methodID := binary.BigEndian.Uint32(input[:4])
	return p.Precompile.RequiredGas(input, p.IsTransaction(methodID))
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(ctx, evm.StateDB, contract, readonly)
	})
}

func (p Precompile) Execute(ctx sdk.Context, stateDB vm.StateDB, contract *vm.Contract, readOnly bool) ([]byte, error) {
	methodID, input, err := cmn.ParseMethod(contract.Input, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	var bz []byte

	switch methodID {
	// Custom transactions
	case ClaimRewardsID:
		return cmn.RunWithStateDB(ctx, p.ClaimRewards, input, stateDB, contract)
	// Distribution transactions
	case SetWithdrawAddressID:
		return cmn.RunWithStateDB(ctx, p.SetWithdrawAddress, input, stateDB, contract)
	case WithdrawDelegatorRewardsID:
		return cmn.RunWithStateDB(ctx, p.WithdrawDelegatorReward, input, stateDB, contract)
	case WithdrawValidatorCommissionID:
		return cmn.RunWithStateDB(ctx, p.WithdrawValidatorCommission, input, stateDB, contract)
	case FundCommunityPoolID:
		bz, err = p.FundCommunityPool(ctx, contract, stateDB, method, args)
	case DepositValidatorRewardsPoolID:
		bz, err = p.DepositValidatorRewardsPool(ctx, contract, stateDB, method, args)
	// Distribution queries
	case ValidatorDistributionInfoID:
		bz, err = p.ValidatorDistributionInfo(ctx, contract, method, args)
	case ValidatorOutstandingRewardsID:
		bz, err = p.ValidatorOutstandingRewards(ctx, contract, method, args)
	case ValidatorCommissionID:
		bz, err = p.ValidatorCommission(ctx, contract, method, args)
	case ValidatorSlashesID:
		bz, err = p.ValidatorSlashes(ctx, contract, method, args)
	case DelegationRewardsID:
		bz, err = p.DelegationRewards(ctx, contract, method, args)
	case DelegationTotalRewardsID:
		bz, err = p.DelegationTotalRewards(ctx, contract, method, args)
	case DelegatorValidatorsID:
		bz, err = p.DelegatorValidators(ctx, contract, method, args)
	case DelegatorWithdrawAddressID:
		bz, err = p.DelegatorWithdrawAddress(ctx, contract, method, args)
	case CommunityPoolID:
		bz, err = p.CommunityPool(ctx, contract, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownID, method.Name)
	}

	return bz, err
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
//
// Available distribution transactions are:
//   - ClaimRewards
//   - SetWithdrawAddress
//   - WithdrawDelegatorReward
//   - WithdrawValidatorCommission
//   - FundCommunityPool
//   - DepositValidatorRewardsPool
func (Precompile) IsTransaction(method uint32) bool {
	switch method {
	case ClaimRewardsID,
		SetWithdrawAddressID,
		WithdrawDelegatorRewardsID,
		WithdrawValidatorCommissionID,
		FundCommunityPoolID,
		DepositValidatorRewardsPoolID:
		return true
	default:
		return false
	}
}
