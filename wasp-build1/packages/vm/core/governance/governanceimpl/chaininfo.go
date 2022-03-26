// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package governanceimpl

import (
	"fmt"

	"github.com/iotaledger/wasp/packages/iscp"
	"github.com/iotaledger/wasp/packages/iscp/assert"
	"github.com/iotaledger/wasp/packages/kv/codec"
	"github.com/iotaledger/wasp/packages/kv/dict"
	"github.com/iotaledger/wasp/packages/kv/kvdecoder"
	"github.com/iotaledger/wasp/packages/vm/core/governance"
)

// getChainInfo view returns general info about the chain: chain ID, chain owner ID, limits and default fees
func getChainInfo(ctx iscp.SandboxView) (dict.Dict, error) {
	info := governance.MustGetChainInfo(ctx.State())
	ret := dict.New()
	ret.Set(governance.VarChainID, codec.EncodeChainID(info.ChainID))
	ret.Set(governance.VarChainOwnerID, codec.EncodeAgentID(info.ChainOwnerID))
	ret.Set(governance.VarDescription, codec.EncodeString(info.Description))
	ret.Set(governance.VarFeeColor, codec.EncodeColor(info.FeeColor))
	ret.Set(governance.VarDefaultOwnerFee, codec.EncodeInt64(info.DefaultOwnerFee))
	ret.Set(governance.VarDefaultValidatorFee, codec.EncodeInt64(info.DefaultValidatorFee))
	ret.Set(governance.VarMaxBlobSize, codec.EncodeUint32(info.MaxBlobSize))
	ret.Set(governance.VarMaxEventSize, codec.EncodeUint16(info.MaxEventSize))
	ret.Set(governance.VarMaxEventsPerReq, codec.EncodeUint16(info.MaxEventsPerReq))

	return ret, nil
}

// setChainInfo sets the configuration parameters of the chain
// Input (all optional):
// - ParamMaxBlobSize         - uint32 maximum size of a blob to be saved in the blob contract.
// - ParamMaxEventSize        - uint16 maximum size of a single event.
// - ParamMaxEventsPerRequest - uint16 maximum number of events per request.
// - ParamOwnerFee            - int64 non-negative value of the owner fee.
// - ParamValidatorFee        - int64 non-negative value of the contract fee.
func setChainInfo(ctx iscp.Sandbox) (dict.Dict, error) {
	a := assert.NewAssert(ctx.Log())
	a.Require(governance.CheckAuthorizationByChainOwner(ctx.State(), ctx.Caller()), "governance.setChainInfo: not authorized")

	params := kvdecoder.New(ctx.Params(), ctx.Log())

	// max blob size
	maxBlobSize := params.MustGetUint32(governance.ParamMaxBlobSize, 0)
	if maxBlobSize > 0 {
		ctx.State().Set(governance.VarMaxBlobSize, codec.Encode(maxBlobSize))
		ctx.Event(fmt.Sprintf("[updated chain config] max blob size: %d", maxBlobSize))
	}

	// max event size
	maxEventSize := params.MustGetUint16(governance.ParamMaxEventSize, 0)
	if maxEventSize > 0 {
		if maxEventSize < governance.MinEventSize {
			// don't allow to set less than MinEventSize to prevent chain owner from bricking the chain
			maxEventSize = governance.MinEventSize
		}
		ctx.State().Set(governance.VarMaxEventSize, codec.Encode(maxEventSize))
		ctx.Event(fmt.Sprintf("[updated chain config] max event size: %d", maxEventSize))
	}

	// max events per request
	maxEventsPerReq := params.MustGetUint16(governance.ParamMaxEventsPerRequest, 0)
	if maxEventsPerReq > 0 {
		if maxEventsPerReq < governance.MinEventsPerRequest {
			maxEventsPerReq = governance.MinEventsPerRequest
		}
		ctx.State().Set(governance.VarMaxEventsPerReq, codec.Encode(maxEventsPerReq))
		ctx.Event(fmt.Sprintf("[updated chain config] max eventsPerRequest: %d", maxEventsPerReq))
	}

	// default owner fee
	ownerFee := params.MustGetInt64(governance.ParamOwnerFee, -1)
	if ownerFee >= 0 {
		ctx.State().Set(governance.VarDefaultOwnerFee, codec.EncodeInt64(ownerFee))
		ctx.Event(fmt.Sprintf("[updated chain config] default owner fee: %d", ownerFee))
	}

	// default validator fee
	validatorFee := params.MustGetInt64(governance.ParamValidatorFee, -1)
	if validatorFee >= 0 {
		ctx.State().Set(governance.VarDefaultValidatorFee, codec.EncodeInt64(validatorFee))
		ctx.Event(fmt.Sprintf("[updated chain config] default validator fee: %d", validatorFee))
	}
	return nil, nil
}

func getMaxBlobSize(ctx iscp.SandboxView) (dict.Dict, error) {
	maxBlobSize, err := ctx.State().Get(governance.VarMaxBlobSize)
	if err != nil {
		ctx.Log().Panicf("error getting max blob size, %v", err)
	}
	ret := dict.New()
	ret.Set(governance.ParamMaxBlobSize, maxBlobSize)
	return ret, nil
}
