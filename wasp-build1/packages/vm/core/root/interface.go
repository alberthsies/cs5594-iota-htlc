package root

import (
	"errors"

	"github.com/iotaledger/wasp/packages/iscp/coreutil"
)

var (
	Contract            = coreutil.NewContract(coreutil.CoreContractRoot, "Root Contract")
	ErrContractNotFound = errors.New("smart contract not found")
)

// state variables
const (
	VarContractRegistry         = "r"
	VarDeployPermissionsEnabled = "a"
	VarDeployPermissions        = "p"
	VarStateInitialized         = "i"
)

// param variables
const (
	ParamDeployer                 = "dp"
	ParamHname                    = "hn"
	ParamName                     = "nm"
	ParamProgramHash              = "ph"
	ParamContractRecData          = "dt"
	ParamContractFound            = "cf"
	ParamDescription              = "ds"
	ParamDeployPermissionsEnabled = "de"
)

// function names
var (
	FuncDeployContract           = coreutil.Func("deployContract")
	FuncGrantDeployPermission    = coreutil.Func("grantDeployPermission")
	FuncRevokeDeployPermission   = coreutil.Func("revokeDeployPermission")
	FuncRequireDeployPermissions = coreutil.Func("requireDeployPermissions")
	FuncFindContract             = coreutil.ViewFunc("findContract")
	FuncGetContractRecords       = coreutil.ViewFunc("getContractRecords")
)
