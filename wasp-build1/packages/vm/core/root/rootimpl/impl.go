// 'root' a core contract on the chain. It is responsible for:
// - initial setup of the chain during chain deployment
// - maintaining of core parameters of the chain
// - maintaining (setting, delegating) chain owner ID
// - maintaining (granting, revoking) smart contract deployment rights
// - deployment of smart contracts on the chain and maintenance of contract registry
package rootimpl

import (
	"fmt"

	"github.com/iotaledger/wasp/packages/iscp"
	"github.com/iotaledger/wasp/packages/iscp/assert"
	"github.com/iotaledger/wasp/packages/kv/codec"
	"github.com/iotaledger/wasp/packages/kv/collections"
	"github.com/iotaledger/wasp/packages/kv/dict"
	"github.com/iotaledger/wasp/packages/kv/kvdecoder"
	"github.com/iotaledger/wasp/packages/vm/core/_default"
	"github.com/iotaledger/wasp/packages/vm/core/accounts"
	"github.com/iotaledger/wasp/packages/vm/core/blob"
	"github.com/iotaledger/wasp/packages/vm/core/blocklog"
	"github.com/iotaledger/wasp/packages/vm/core/governance"
	"github.com/iotaledger/wasp/packages/vm/core/root"
)

var Processor = root.Contract.Processor(initialize,
	root.FuncDeployContract.WithHandler(deployContract),
	root.FuncGrantDeployPermission.WithHandler(grantDeployPermission),
	root.FuncRevokeDeployPermission.WithHandler(revokeDeployPermission),
	root.FuncFindContract.WithHandler(findContract),
	root.FuncGetContractRecords.WithHandler(getContractRecords),
	root.FuncRequireDeployPermissions.WithHandler(requireDeployPermissions),
)

// initialize handles constructor, the "init" request. This is the first call to the chain
// if it fails, chain is not initialized. Does the following:
// - stores chain ID and chain description in the state
// - sets state ownership to the caller
// - creates record in the registry for the 'root' itself
// - deploys other core contracts: 'accounts', 'blob', 'blocklog' by creating records in the registry and calling constructors
// Input:
// - ParamChainID iscp.ChainID. ID of the chain. Cannot be changed
// - ParamChainColor ledgerstate.Color
// - ParamDescription string defaults to "N/A"
// - ParamFeeColor ledgerstate.Color fee color code. Defaults to IOTA color. It cannot be changed
func initialize(ctx iscp.Sandbox) (dict.Dict, error) {
	ctx.Log().Debugf("root.initialize.begin")
	state := ctx.State()
	a := assert.NewAssert(ctx.Log())

	a.Require(state.MustGet(root.VarStateInitialized) == nil, "root.initialize.fail: already initialized")
	a.Require(ctx.Caller().Hname() == 0, "root.init.fail: chain deployer can't be another smart contract")

	contractRegistry := collections.NewMap(state, root.VarContractRegistry)
	a.Require(contractRegistry.MustLen() == 0, "root.initialize.fail: registry not empty")

	mustStoreContract(ctx, _default.Contract, a)
	mustStoreContract(ctx, root.Contract, a)
	mustStoreAndInitCoreContract(ctx, blob.Contract, a)
	mustStoreAndInitCoreContract(ctx, accounts.Contract, a)
	mustStoreAndInitCoreContract(ctx, blocklog.Contract, a)
	govParams := ctx.Params().Clone()
	govParams.Set(governance.ParamChainOwner, ctx.Caller().Bytes()) // chain owner is whoever sends init request
	mustStoreAndInitCoreContract(ctx, governance.Contract, a, govParams)

	state.Set(root.VarDeployPermissionsEnabled, codec.EncodeBool(true))
	state.Set(root.VarStateInitialized, []byte{0xFF})

	ctx.Log().Debugf("root.initialize.success")
	return nil, nil
}

// deployContract deploys contract and calls its 'init' constructor.
// If call to the constructor returns an error or an other error occurs,
// removes smart contract form the registry as if it was never attempted to deploy
// Inputs:
// - ParamName string, the unique name of the contract in the chain. Later used as hname
// - ParamProgramHash HashValue is a hash of the blob which represents program binary in the 'blob' contract.
//     In case of hardcoded examples its an arbitrary unique hash set in the global call examples.AddProcessor
// - ParamDescription string is an arbitrary string. Defaults to "N/A"
func deployContract(ctx iscp.Sandbox) (dict.Dict, error) {
	ctx.Log().Debugf("root.deployContract.begin")
	if !isAuthorizedToDeploy(ctx) {
		return nil, fmt.Errorf("root.deployContract: deploy not permitted for: %s", ctx.Caller())
	}
	params := kvdecoder.New(ctx.Params(), ctx.Log())
	a := assert.NewAssert(ctx.Log())

	progHash := params.MustGetHashValue(root.ParamProgramHash)
	description := params.MustGetString(root.ParamDescription, "N/A")
	name := params.MustGetString(root.ParamName)
	a.Require(name != "", "wrong name")

	// pass to init function all params not consumed so far
	initParams := dict.New()
	for key, value := range ctx.Params() {
		if key != root.ParamProgramHash && key != root.ParamName && key != root.ParamDescription {
			initParams.Set(key, value)
		}
	}
	// call to load VM from binary to check if it loads successfully
	err := ctx.DeployContract(progHash, "", "", nil)
	a.Require(err == nil, "root.deployContract.fail 1: %v", err)

	// VM loaded successfully. Storing contract in the registry and calling constructor
	mustStoreContractRecord(ctx, &root.ContractRecord{
		ProgramHash: progHash,
		Description: description,
		Name:        name,
		Creator:     ctx.Caller(),
	}, a)
	_, err = ctx.Call(iscp.Hn(name), iscp.EntryPointInit, initParams, nil)
	a.RequireNoError(err)

	ctx.Event(fmt.Sprintf("[deploy] name: %s hname: %s, progHash: %s, dscr: '%s'",
		name, iscp.Hn(name), progHash.String(), description))
	return nil, nil
}

// findContract view finds and returns encoded record of the contract
// Input:
// - ParamHname
// Output:
// - ParamData
func findContract(ctx iscp.SandboxView) (dict.Dict, error) {
	params := kvdecoder.New(ctx.Params())
	hname, err := params.GetHname(root.ParamHname)
	if err != nil {
		return nil, err
	}
	rec, found := root.FindContract(ctx.State(), hname)
	ret := dict.New()
	ret.Set(root.ParamContractRecData, rec.Bytes())
	var foundByte [1]byte
	if found {
		foundByte[0] = 0xFF
	}
	ret.Set(root.ParamContractFound, foundByte[:])
	return ret, nil
}

// grantDeployPermission grants permission to deploy contracts
// Input:
//  - ParamDeployer iscp.AgentID
func grantDeployPermission(ctx iscp.Sandbox) (dict.Dict, error) {
	a := assert.NewAssert(ctx.Log())
	a.Require(isChainOwner(a, ctx), "root.grantDeployPermissions: not authorized")

	params := kvdecoder.New(ctx.Params(), ctx.Log())
	deployer := params.MustGetAgentID(root.ParamDeployer)

	collections.NewMap(ctx.State(), root.VarDeployPermissions).MustSetAt(deployer.Bytes(), []byte{0xFF})
	ctx.Event(fmt.Sprintf("[grant deploy permission] to agentID: %s", deployer))
	return nil, nil
}

// revokeDeployPermission revokes permission to deploy contracts
// Input:
//  - ParamDeployer iscp.AgentID
func revokeDeployPermission(ctx iscp.Sandbox) (dict.Dict, error) {
	a := assert.NewAssert(ctx.Log())
	a.Require(isChainOwner(a, ctx), "root.revokeDeployPermissions: not authorized")

	params := kvdecoder.New(ctx.Params(), ctx.Log())
	deployer := params.MustGetAgentID(root.ParamDeployer)

	collections.NewMap(ctx.State(), root.VarDeployPermissions).MustDelAt(deployer.Bytes())
	ctx.Event(fmt.Sprintf("[revoke deploy permission] from agentID: %s", deployer))
	return nil, nil
}

func getContractRecords(ctx iscp.SandboxView) (dict.Dict, error) {
	src := collections.NewMapReadOnly(ctx.State(), root.VarContractRegistry)

	ret := dict.New()
	dst := collections.NewMap(ret, root.VarContractRegistry)
	src.MustIterate(func(elemKey []byte, value []byte) bool {
		dst.MustSetAt(elemKey, value)
		return true
	})

	return ret, nil
}

func requireDeployPermissions(ctx iscp.Sandbox) (dict.Dict, error) {
	a := assert.NewAssert(ctx.Log())
	a.Require(isChainOwner(a, ctx), "root.revokeDeployPermissions: not authorized")
	params := kvdecoder.New(ctx.Params())
	a.Require(ctx.Params().MustHas(root.ParamDeployPermissionsEnabled), "root.revokeDeployPermissions: ParamDeployPermissionsEnabled missing")
	permissionsEnabled := params.MustGetBool(root.ParamDeployPermissionsEnabled)
	ctx.State().Set(root.VarDeployPermissionsEnabled, codec.EncodeBool(permissionsEnabled))
	return nil, nil
}
