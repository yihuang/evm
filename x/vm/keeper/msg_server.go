package keeper

import (
	"context"
	"cosmossdk.io/x/feegrant"
	"encoding/hex"
	"encoding/json"
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/types/registry"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"strconv"

	"github.com/hashicorp/go-metrics"

	cmttypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/evm/x/vm/types"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var _ types.MsgServer = &Keeper{}

// EthereumTx implements the gRPC MsgServer interface. It receives a transaction which is then
// executed (i.e applied) against the go-ethereum EVM. The provided SDK Context is set to the Keeper
// so that it can implements and call the StateDB methods without receiving it as a function
// parameter.
func (k *Keeper) EthereumTx(goCtx context.Context, msg *types.MsgEthereumTx) (*types.MsgEthereumTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender := msg.From
	tx := msg.AsTransaction()
	txIndex := k.GetTxIndexTransient(ctx)

	labels := []metrics.Label{
		telemetry.NewLabel("tx_type", fmt.Sprintf("%d", tx.Type())),
	}
	if tx.To() == nil {
		labels = append(labels, telemetry.NewLabel("execution", "create"))
	} else {
		labels = append(labels, telemetry.NewLabel("execution", "call"))
	}

	response, err := k.ApplyTransaction(ctx, tx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to apply transaction")
	}

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "ethereum_tx", "total"},
			1,
			labels,
		)

		if response.GasUsed != 0 {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "ethereum_tx", "gas_used", "total"},
				float32(response.GasUsed),
				labels,
			)

			// Observe which users define a gas limit >> gas used. Note, that
			// gas_limit and gas_used are always > 0
			gasLimit := math.LegacyNewDec(int64(tx.Gas()))                        //#nosec G115 -- int overflow is not a concern here -- tx gas is not going to exceed int64 max value
			gasRatio, err := gasLimit.QuoInt64(int64(response.GasUsed)).Float64() //#nosec G115 -- int overflow is not a concern here -- gas used is not going to exceed int64 max value
			if err == nil {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "ethereum_tx", "gas_limit", "per", "gas_used"},
					float32(gasRatio),
					labels,
				)
			}
		}
	}()

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyAmount, tx.Value().String()),
		// add event for ethereum transaction hash format
		sdk.NewAttribute(types.AttributeKeyEthereumTxHash, response.Hash),
		// add event for index of valid ethereum tx
		sdk.NewAttribute(types.AttributeKeyTxIndex, strconv.FormatUint(txIndex, 10)),
		// add event for eth tx gas used, we can't get it from cosmos tx result when it contains multiple eth tx msgs.
		sdk.NewAttribute(types.AttributeKeyTxGasUsed, strconv.FormatUint(response.GasUsed, 10)),
	}

	if len(ctx.TxBytes()) > 0 {
		// add event for tendermint transaction hash format
		hash := cmttypes.Tx(ctx.TxBytes()).Hash()
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyTxHash, hex.EncodeToString(hash)))
	}

	if to := tx.To(); to != nil {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyRecipient, to.Hex()))
	}

	if response.Failed() {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyEthereumTxFailed, response.VmError))
	}

	txLogAttrs := make([]sdk.Attribute, len(response.Logs))
	for i, log := range response.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to encode log")
		}
		txLogAttrs[i] = sdk.NewAttribute(types.AttributeKeyTxLog, string(value))
	}

	// emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeEthereumTx,
			attrs...,
		),
		sdk.NewEvent(
			types.EventTypeTxLog,
			txLogAttrs...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, sender),
			sdk.NewAttribute(types.AttributeKeyTxType, fmt.Sprintf("%d", tx.Type())),
		),
	})

	return response, nil
}

// UpdateParams implements the gRPC MsgServer interface. When an UpdateParams
// proposal passes, it updates the module parameters. The update can only be
// performed if the requested authority is the Cosmos SDK governance module
// account.
func (k *Keeper) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if k.authority.String() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority, expected %s, got %s", k.authority.String(), req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k *Keeper) MigrateAccount(goCtx context.Context, req *types.MsgMigrateAccount) (*types.MsgMigrateAccountResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// todo: should someone need to sign with both keys?
	// one key signing is probably fine - this functions with similar effect to bank send.
	// on the other hand, may be good for users to confirm that the other address is actually functional;
	// hmm

	originalAddress, err := sdk.AccAddressFromBech32(req.OriginalAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid original address: %s", err)
	}

	newAddress, err := sdk.AccAddressFromBech32(req.NewAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid new address: %s", err)
	}

	// === Bank ===

	// todo: paginate this as there can be a large number of coins and that could exceed the number of gas that the block supports
	// scope out the max, average number of tokens that accounts hold
	// this will never be a ddos vector, but we want to make sure that the normal upper case is properly handled
	balances := k.bankWrapper.GetAllBalances(ctx, originalAddress)

	err = k.bankWrapper.SendCoins(ctx, originalAddress, newAddress, balances)
	if err != nil {
		return nil, err
	}

	// === Staking ===

	// todo: paginate and select the proper number of maximal retrievals
	delegations, err := k.stakingKeeper.GetDelegatorDelegations(ctx, originalAddress, 10)

	for _, delegation := range delegations {
		// Seems like we don't need to go through the whole unbond -> send to user -> send to new address -> bond
		// and that we can remove / reset the delegation instead. Tokens aren't moving anywhere anyways
		err = k.stakingKeeper.RemoveDelegation(ctx, delegation)
		if err != nil {
			return nil, err
		}

		err = k.stakingKeeper.SetDelegation(ctx, stakingtypes.Delegation{
			DelegatorAddress: newAddress.String(),
			ValidatorAddress: delegation.ValidatorAddress,
			Shares:           delegation.Shares,
		})
		if err != nil {
			return nil, err
		}

		// todo: handle vesting accounts and migrate only amounts that are not in the process of vesting
	}

	// === Feegrant ===
	// todo: paginate and select the proper number of maximal retrievals
	allowancesResponse, err := k.feegrantKeeper.AllowancesByGranter(ctx, &feegrant.QueryAllowancesByGranterRequest{
		Granter: originalAddress.String(),
		Pagination: &query.PageRequest{
			Key:        nil,
			Offset:     0,
			Limit:      0,
			CountTotal: false,
			Reverse:    false,
		},
	})
	if err != nil {
		return nil, err
	}

	allowances := allowancesResponse.Allowances

	for _, allowance := range allowances {
		k.feegrantKeeper
	}

	// === Authz ===
	// todo: paginate and select the proper number of maximal retrievals
	authzGrantsResponse, err := k.authzKeeper.GranterGrants(ctx, &authz.QueryGranterGrantsRequest{
		Granter: originalAddress.String(),
		Pagination: &query.PageRequest{
			Key:        nil,
			Offset:     0,
			Limit:      0,
			CountTotal: false,
			Reverse:    false,
		},
	})
	if err != nil {
		return nil, err
	}

	authzGrants := authzGrantsResponse.Grants

	for _, grant := range authzGrants {
		granteeAddress, err := sdk.AccAddressFromBech32(grant.Granter)
		if err != nil {
			return nil, err
		}

		granterAddress, err := sdk.AccAddressFromBech32(grant.Granter)
		if err != nil {
			return nil, err
		}

		// todo: is the message type string the same as the type url?
		err = k.authzKeeper.DeleteGrant(ctx, granteeAddress, granterAddress, grant.Authorization.TypeUrl)
		if err != nil {
			return nil, err
		}

		var authorization authz.Authorization

		// todo: test this
		err = k.cdc.UnpackAny(grant.Authorization, &authorization)
		if err != nil {
			return nil, fmt.Errorf("unknown message type: %s", grant.Authorization.TypeUrl)
		}

		err = k.authzKeeper.SaveGrant(ctx, granteeAddress, granterAddress, authorization, grant.Expiration)
		if err != nil {
			return nil, err
		}
	}

	return &types.MsgMigrateAccountResponse{}, nil
}
