// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package gotemplates

var eventsGo = map[string]string{
	// *******************************
	"events.go": `
//nolint:gocritic
$#emit goHeader
$#emit importWasmTypes

$#set TypeName $Package$+Events
type $TypeName struct {
}
$#each events eventFunc
`,
	// *******************************
	"eventFunc": `
$#set separator 
$#set params 
$#each event eventParam

func (e $TypeName) $EvtName($params) {
	evt := wasmlib.NewEventEncoder("$package.$evtName")
$#each event eventEmit
	evt.Emit()
}
`,
	// *******************************
	"eventParam": `
$#set params $params$separator$fldName $fldLangType
$#set separator , 
`,
	// *******************************
	"eventEmit": `
	evt.Encode(wasmtypes.$FldType$+ToString($fldName))
`,
}
