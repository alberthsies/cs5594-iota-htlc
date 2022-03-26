// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package gotemplates

var libGo = map[string]string{
	// *******************************
	"lib.go": `
//nolint:dupl
$#emit goHeader

var exportMap = wasmlib.ScExportMap{
	Names: []string{
$#each func libExportName
	},
	Funcs: []wasmlib.ScFuncContextFunction{
$#each func libExportFunc
	},
	Views: []wasmlib.ScViewContextFunction{
$#each func libExportView
	},
}

func OnLoad(index int32) {
	if index >= 0 {
		wasmlib.ScExportsCall(index, &exportMap)
		return
	}

	wasmlib.ScExportsExport(&exportMap)
}
$#each func libThunk
`,
	// *******************************
	"libExportName": `
    	$Kind$FuncName,
`,
	// *******************************
	"libExportFunc": `
$#if func libExportFuncThunk
`,
	// *******************************
	"libExportFuncThunk": `
    	$kind$FuncName$+Thunk,
`,
	// *******************************
	"libExportView": `
$#if view libExportViewThunk
`,
	// *******************************
	"libExportViewThunk": `
    	$kind$FuncName$+Thunk,
`,
	// *******************************
	"libThunk": `

type $FuncName$+Context struct {
$#if func PackageEvents
$#if param ImmutableFuncNameParams
$#if result MutableFuncNameResults
$#if func MutablePackageState
$#if view ImmutablePackageState
}

func $kind$FuncName$+Thunk(ctx wasmlib.Sc$Kind$+Context) {
	ctx.Log("$package.$kind$FuncName")
$#if result initResultDict
	f := &$FuncName$+Context{
$#if param ImmutableFuncNameParamsInit
$#if result MutableFuncNameResultsInit
$#if func MutablePackageStateInit
$#if view ImmutablePackageStateInit
	}
$#emit accessCheck
$#each mandatory requireMandatory
	$kind$FuncName(ctx, f)
$#if result returnResultDict
	ctx.Log("$package.$kind$FuncName ok")
}
`,
	// *******************************
	"initResultDict": `
	results := wasmlib.NewScDict()
`,
	// *******************************
	"PackageEvents": `
$#if events PackageEventsExist
`,
	// *******************************
	"PackageEventsExist": `
	Events  $Package$+Events
`,
	// *******************************
	"ImmutableFuncNameParams": `
	Params  Immutable$FuncName$+Params
`,
	// *******************************
	"ImmutableFuncNameParamsInit": `
		Params: Immutable$FuncName$+Params{
			proxy: wasmlib.NewParamsProxy(),
		},
`,
	// *******************************
	"MutableFuncNameResults": `
	Results Mutable$FuncName$+Results
`,
	// *******************************
	"MutableFuncNameResultsInit": `
		Results: Mutable$FuncName$+Results{
			proxy: results.AsProxy(),
		},
`,
	// *******************************
	"MutablePackageState": `
	State   Mutable$Package$+State
`,
	// *******************************
	"MutablePackageStateInit": `
		State: Mutable$Package$+State{
			proxy: wasmlib.NewStateProxy(),
		},
`,
	// *******************************
	"ImmutablePackageState": `
	State   Immutable$Package$+State
`,
	// *******************************
	"ImmutablePackageStateInit": `
		State: Immutable$Package$+State{
			proxy: wasmlib.NewStateProxy(),
		},
`,
	// *******************************
	"returnResultDict": `
	ctx.Results(results)
`,
	// *******************************
	"requireMandatory": `
	ctx.Require(f.Params.$FldName().Exists(), "missing mandatory $fldName")
`,
	// *******************************
	"accessCheck": `
$#set accessFinalize accessOther
$#emit caseAccess$funcAccess
$#emit $accessFinalize
`,
	// *******************************
	"caseAccess": `
$#set accessFinalize accessDone
`,
	// *******************************
	"caseAccessself": `
$#if funcAccessComment accessComment
	ctx.Require(ctx.Caller() == ctx.AccountID(), "no permission")

$#set accessFinalize accessDone
`,
	// *******************************
	"caseAccesschain": `
$#if funcAccessComment accessComment
	ctx.Require(ctx.Caller() == ctx.ChainOwnerID(), "no permission")

$#set accessFinalize accessDone
`,
	// *******************************
	"caseAccesscreator": `
$#if funcAccessComment accessComment
	ctx.Require(ctx.Caller() == ctx.ContractCreator(), "no permission")

$#set accessFinalize accessDone
`,
	// *******************************
	"accessOther": `
$#if funcAccessComment accessComment
	access := f.State.$FuncAccess()
	ctx.Require(access.Exists(), "access not set: $funcAccess")
	ctx.Require(ctx.Caller() == access.Value(), "no permission")

`,
	// *******************************
	"accessDone": `
`,
	// *******************************
	"accessComment": `

	$funcAccessComment
`,
}
