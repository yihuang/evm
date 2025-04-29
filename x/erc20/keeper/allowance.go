package keeper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/evm/x/erc20/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

// GetAllowance returns the allowance of the given owner and spender
// on the given erc20 precompile address.
func (k Keeper) GetAllowance(
	ctx sdk.Context,
	erc20 common.Address,
	owner common.Address,
	spender common.Address,
) (*big.Int, error) {
	allowanceKey := types.AllowanceKey(erc20, owner, spender)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAllowance)

	var allowance types.Allowance
	bz := store.Get(allowanceKey)
	if bz == nil {
		return common.Big0, nil
	}

	k.cdc.MustUnmarshal(bz, &allowance)

	return allowance.Value.BigInt(), nil
}

// SetAllowance sets the allowance of the given owner and spender
// on the given erc20 precompile address.
func (k Keeper) SetAllowance(
	ctx sdk.Context,
	erc20 common.Address,
	owner common.Address,
	spender common.Address,
	value *big.Int,
) error {
	return k.SetAllowanceWithValidation(ctx, erc20, owner, spender, value, false)
}

// DeleteAllowance deletes the allowance of the given owner and spender
// on the given erc20 precompile address.
func (k Keeper) DeleteAllowance(
	ctx sdk.Context,
	erc20 common.Address,
	owner common.Address,
	spender common.Address,
) error {
	return k.SetAllowanceWithValidation(ctx, erc20, owner, spender, common.Big0, false)
}

// SetAllowanceWithValidation sets the allowance of the given owner and spender with validation.
// If allowDisabledTokenPair is true, it allows setting allowance for disabled token pairs.
func (k Keeper) SetAllowanceWithValidation(
	ctx sdk.Context,
	erc20 common.Address,
	owner common.Address,
	spender common.Address,
	value *big.Int,
	allowDisabledTokenPair bool,
) error {
	tokenPairID := k.GetERC20Map(ctx, erc20)
	tokenPair, found := k.GetTokenPair(ctx, tokenPairID)
	if !found {
		return errorsmod.Wrapf(
			types.ErrTokenPairNotFound, "token pair for address '%s' not registered", erc20,
		)
	}
	if !allowDisabledTokenPair && !tokenPair.Enabled {
		return errorsmod.Wrapf(
			types.ErrERC20TokenPairDisabled, "token pair for address '%s' is disabled", erc20,
		)
	}

	if (owner == common.Address{}) {
		return errorsmod.Wrapf(errortypes.ErrInvalidAddress, "erc20 address is empty")
	}
	if (spender == common.Address{}) {
		return errorsmod.Wrapf(errortypes.ErrInvalidAddress, "spender address is empty")
	}

	switch {
	case value == nil || value.Sign() == 0:
		// case 1. value nil or zero -> delete allowance
		k.setAllowance(ctx, erc20, owner, spender, nil)
	case value.Sign() < 0:
		// case 2. value negative -> return error
		return errorsmod.Wrapf(types.ErrInvalidAllowance, "value '%s' is less than zero", value)
	case value.BitLen() > 256:
		// case 3. value greater than max value of uint256 -> return error
		return errorsmod.Wrapf(types.ErrInvalidAllowance, "value '%s' is greater than max value of uint256", value)
	default:
		k.setAllowance(ctx, erc20, owner, spender, value)
	}

	return nil
}

func (k Keeper) setAllowance(
	ctx sdk.Context,
	erc20 common.Address,
	owner common.Address,
	spender common.Address,
	value *big.Int,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAllowance)
	allowanceKey := types.AllowanceKey(erc20, owner, spender)

	if value == nil {
		store.Delete(allowanceKey)
	} else {
		allowance := types.NewAllowance(erc20, owner, spender, value)
		bz := k.cdc.MustMarshal(&allowance)
		store.Set(allowanceKey, bz)
	}
}

// GetAllowances returns all allowances stored on the given erc20 precompile address.
func (k Keeper) GetAllowances(
	ctx sdk.Context,
) []types.Allowance {
	allowances := []types.Allowance{}

	k.IterateAllowances(ctx, func(allowance types.Allowance) (stop bool) {
		allowances = append(allowances, allowance)
		return false
	})

	return allowances
}

// IterateAllowances iterates through all allowances stored on the given erc20 precompile address.
func (k Keeper) IterateAllowances(
	ctx sdk.Context,
	cb func(allowance types.Allowance) (stop bool),
) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, types.KeyPrefixAllowance)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var allowance types.Allowance
		k.cdc.MustUnmarshal(iterator.Value(), &allowance)

		if cb(allowance) {
			break
		}
	}
}

func (k Keeper) deleteAllowances(ctx sdk.Context, erc20 common.Address) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixAllowance)
	iterator := storetypes.KVStorePrefixIterator(store, erc20.Bytes())
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}
