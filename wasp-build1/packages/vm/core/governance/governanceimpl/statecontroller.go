// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package governanceimpl

import (
	"fmt"

	"github.com/iotaledger/wasp/packages/iscp"
	"github.com/iotaledger/wasp/packages/iscp/assert"
	"github.com/iotaledger/wasp/packages/iscp/coreutil"
	"github.com/iotaledger/wasp/packages/kv/collections"
	"github.com/iotaledger/wasp/packages/kv/dict"
	"github.com/iotaledger/wasp/packages/kv/kvdecoder"
	"github.com/iotaledger/wasp/packages/vm/core/blocklog"
	"github.com/iotaledger/wasp/packages/vm/core/governance"
)

// rotateStateController the entry point is called when committee is about to be rotated to the new address
// If it fails, nothing happens and the state has trace of the failure in the state
// If it is successful VM takes over and replaces resulting transaction with
// governance transition. The state of the chain remains unchanged
func rotateStateController(ctx iscp.Sandbox) (dict.Dict, error) {
	a := assert.NewAssert(ctx.Log())
	a.RequireChainOwner(ctx, "rotateStateController")
	par := kvdecoder.New(ctx.Params(), ctx.Log())
	newStateControllerAddr := par.MustGetAddress(governance.ParamStateControllerAddress)
	// check is address is allowed
	amap := collections.NewMapReadOnly(ctx.State(), governance.StateVarAllowedStateControllerAddresses)
	a.Require(amap.MustHasAt(newStateControllerAddr.Bytes()), "rotateStateController: address is not allowed as next state address: %s", newStateControllerAddr.Base58())

	if !newStateControllerAddr.Equals(ctx.StateAnchor().StateAddress()) {
		// rotate request to another address has been issued. State update will be taken over by VM and will have no effect
		// By setting StateVarRotateToAddress we signal the VM this special situation
		// StateVarRotateToAddress value should never persist in the state
		ctx.Log().Infof("Governance::RotateStateController: newStateControllerAddress=%s", newStateControllerAddr.String())
		ctx.State().Set(governance.StateVarRotateToAddress, newStateControllerAddr.Bytes())
		return nil, nil
	}
	// here the new state controller address from the request equals to the state controller address in the anchor output
	// Two situations possible:
	// - either there's no need to rotate
	// - or it just has been rotated. In case of the second situation we emit a 'rotate' event
	addrs, err := ctx.Call(coreutil.CoreContractBlocklogHname, blocklog.FuncControlAddresses.Hname(), nil, nil)
	a.RequireNoError(err)
	par = kvdecoder.New(addrs, ctx.Log())
	storedStateController := par.MustGetAddress(blocklog.ParamStateControllerAddress)
	if !storedStateController.Equals(newStateControllerAddr) {
		// state controller address recorded in the blocklog is different from the new one
		// It means rotation happened
		ctx.Event(fmt.Sprintf("rotate %s %s", newStateControllerAddr.Base58(), storedStateController.Base58()))
		return nil, nil
	}
	// no need to rotate because address does not change
	return nil, nil
}

func addAllowedStateControllerAddress(ctx iscp.Sandbox) (dict.Dict, error) {
	a := assert.NewAssert(ctx.Log())
	a.RequireChainOwner(ctx, "addAllowedStateControllerAddress")
	par := kvdecoder.New(ctx.Params(), ctx.Log())
	addr := par.MustGetAddress(governance.ParamStateControllerAddress)
	amap := collections.NewMap(ctx.State(), governance.StateVarAllowedStateControllerAddresses)
	amap.MustSetAt(addr.Bytes(), []byte{0xFF})
	return nil, nil
}

func removeAllowedStateControllerAddress(ctx iscp.Sandbox) (dict.Dict, error) {
	a := assert.NewAssert(ctx.Log())
	a.RequireChainOwner(ctx, "removeAllowedStateControllerAddress")
	par := kvdecoder.New(ctx.Params(), ctx.Log())
	addr := par.MustGetAddress(governance.ParamStateControllerAddress)
	amap := collections.NewMap(ctx.State(), governance.StateVarAllowedStateControllerAddresses)
	amap.MustDelAt(addr.Bytes())
	return nil, nil
}

func getAllowedStateControllerAddresses(ctx iscp.SandboxView) (dict.Dict, error) {
	amap := collections.NewMapReadOnly(ctx.State(), governance.StateVarAllowedStateControllerAddresses)
	if amap.MustLen() == 0 {
		return nil, nil
	}
	ret := dict.New()
	retArr := collections.NewArray16(ret, governance.ParamAllowedStateControllerAddresses)
	amap.MustIterateKeys(func(elemKey []byte) bool {
		retArr.MustPush(elemKey)
		return true
	})
	return ret, nil
}
