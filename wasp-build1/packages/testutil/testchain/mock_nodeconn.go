package testchain

import (
	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/wasp/packages/chain"
	"github.com/iotaledger/wasp/packages/metrics/nodeconnmetrics"
)

type MockedNodeConn struct {
	id                              string
	onPullState                     func()
	onPullTransactionInclusionState func(txid ledgerstate.TransactionID)
	onPullConfirmedOutput           func(outputID ledgerstate.OutputID)
	onPostTransaction               func(tx *ledgerstate.Transaction)
}

var _ chain.ChainNodeConnection = &MockedNodeConn{}

func NewMockedNodeConnection(id string) *MockedNodeConn {
	return &MockedNodeConn{id: id}
}

func (m *MockedNodeConn) ID() string {
	return m.id
}

func (m *MockedNodeConn) PullState() {
	m.onPullState()
}

func (m *MockedNodeConn) PullTransactionInclusionState(txid ledgerstate.TransactionID) {
	m.onPullTransactionInclusionState(txid)
}

func (m *MockedNodeConn) PullConfirmedOutput(outputID ledgerstate.OutputID) {
	m.onPullConfirmedOutput(outputID)
}

func (m *MockedNodeConn) PostTransaction(tx *ledgerstate.Transaction) {
	m.onPostTransaction(tx)
}

func (m *MockedNodeConn) OnPullState(f func()) {
	m.onPullState = f
}

func (m *MockedNodeConn) OnPullTransactionInclusionState(f func(txid ledgerstate.TransactionID)) {
	m.onPullTransactionInclusionState = f
}

func (m *MockedNodeConn) OnPullConfirmedOutput(f func(outputID ledgerstate.OutputID)) {
	m.onPullConfirmedOutput = f
}

func (m *MockedNodeConn) OnPostTransaction(f func(tx *ledgerstate.Transaction)) {
	m.onPostTransaction = f
}

func (m *MockedNodeConn) AttachToTransactionReceived(chain.NodeConnectionHandleTransactionFun) {}
func (m *MockedNodeConn) AttachToInclusionStateReceived(chain.NodeConnectionHandleInclusionStateFun) {
}
func (m *MockedNodeConn) AttachToOutputReceived(chain.NodeConnectionHandleOutputFun) {}
func (m *MockedNodeConn) AttachToUnspentAliasOutputReceived(chain.NodeConnectionHandleUnspentAliasOutputFun) {
}

func (m *MockedNodeConn) DetachFromTransactionReceived()        {}
func (m *MockedNodeConn) DetachFromInclusionStateReceived()     {}
func (m *MockedNodeConn) DetachFromOutputReceived()             {}
func (m *MockedNodeConn) DetachFromUnspentAliasOutputReceived() {}

func (m *MockedNodeConn) Close() {}

func (m *MockedNodeConn) GetMetrics() nodeconnmetrics.NodeConnectionMessagesMetrics {
	return nodeconnmetrics.NewEmptyNodeConnectionMessagesMetrics()
}
