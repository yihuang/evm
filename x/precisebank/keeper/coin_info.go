package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/evm/x/precisebank/types"
)

func (k Keeper) GetCoinInfo(goCtx context.Context) types.CoinInfo {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := store.Get(types.CoinInfoKey)
	if len(bz) == 0 {
		return types.DefaultCoinInfo()
	}
	var coinInfo types.CoinInfo
	k.cdc.MustUnmarshal(bz, &coinInfo)
	return coinInfo
}

func (k Keeper) SetCoinInfo(goCtx context.Context, coinInfo types.CoinInfo) error {
	sdkCtx := sdk.UnwrapSDKContext(goCtx)
	store := sdkCtx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&coinInfo)
	store.Set(types.CoinInfoKey, bz)
	return nil
}
