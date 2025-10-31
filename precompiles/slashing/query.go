package slashing

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/types"
)

const (
	// GetSigningInfoMethod defines the ABI method name for the slashing SigningInfo query
	GetSigningInfoMethod = "getSigningInfo"
	// GetSigningInfosMethod defines the ABI method name for the slashing SigningInfos query
	GetSigningInfosMethod = "getSigningInfos"
	// GetParamsMethod defines the ABI method name for the slashing Params query
	GetParamsMethod = "getParams"
)

// GetSigningInfo handles the `getSigningInfo` precompile call.
// It expects a single argument: the validator's consensus address in hex format.
// That address comes from the validator's CometBFT ed25519 public key,
// typically found in `$HOME/.evmd/config/priv_validator_key.json`.
func (p *Precompile) GetSigningInfo(
	ctx sdk.Context,
	args *GetSigningInfoCall,
) (*GetSigningInfoReturn, error) {
	req, err := ParseSigningInfoArgs(*args, p.consCodec)
	if err != nil {
		return nil, err
	}

	res, err := p.slashingKeeper.SigningInfo(ctx, req)
	if err != nil {
		return nil, err
	}

	out, err := new(SigningInfoOutput).FromResponse(res)
	if err != nil {
		return nil, err
	}

	// Convert to generated SigningInfo type
	return &GetSigningInfoReturn{
		SigningInfo: SigningInfo{
			ValidatorAddress:    out.SigningInfo.ValidatorAddress,
			StartHeight:         out.SigningInfo.StartHeight,
			IndexOffset:         out.SigningInfo.IndexOffset,
			JailedUntil:         out.SigningInfo.JailedUntil,
			Tombstoned:          out.SigningInfo.Tombstoned,
			MissedBlocksCounter: out.SigningInfo.MissedBlocksCounter,
		},
	}, nil
}

// GetSigningInfos implements the query to get signing info for all validators.
func (p *Precompile) GetSigningInfos(
	ctx sdk.Context,
	args *GetSigningInfosCall,
) (*GetSigningInfosReturn, error) {
	method := &abi.Method{}
	req, err := ParseSigningInfosArgs(method, *args)
	if err != nil {
		return nil, err
	}

	res, err := p.slashingKeeper.SigningInfos(ctx, req)
	if err != nil {
		return nil, err
	}

	out, err := new(SigningInfosOutput).FromResponse(res)
	if err != nil {
		return nil, err
	}

	// Convert to generated types
	signingInfos := make([]SigningInfo, len(out.SigningInfos))
	for i, info := range out.SigningInfos {
		signingInfos[i] = SigningInfo{
			ValidatorAddress:    info.ValidatorAddress,
			StartHeight:         info.StartHeight,
			IndexOffset:         info.IndexOffset,
			JailedUntil:         info.JailedUntil,
			Tombstoned:          info.Tombstoned,
			MissedBlocksCounter: info.MissedBlocksCounter,
		}
	}

	return &GetSigningInfosReturn{
		SigningInfos: signingInfos,
		PageResponse: PageResponse{
			NextKey: out.PageResponse.NextKey,
			Total:   out.PageResponse.Total,
		},
	}, nil
}

// GetParams implements the query to get the slashing parameters.
func (p *Precompile) GetParams(
	ctx sdk.Context,
	_ *GetParamsCall,
) (*GetParamsReturn, error) {
	res, err := p.slashingKeeper.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	out := new(ParamsOutput).FromResponse(res)

	// Convert to generated Params type
	return &GetParamsReturn{
		Params: Params{
			SignedBlocksWindow:      out.Params.SignedBlocksWindow,
			MinSignedPerWindow:      out.Params.MinSignedPerWindow,
			DowntimeJailDuration:    out.Params.DowntimeJailDuration,
			SlashFractionDoubleSign: out.Params.SlashFractionDoubleSign,
			SlashFractionDowntime:   out.Params.SlashFractionDowntime,
		},
	}, nil
}
