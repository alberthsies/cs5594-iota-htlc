// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package chain

import (
	"fmt"
	"time"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/iotaledger/hive.go/events"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/wasp/packages/chain/messages"
	"github.com/iotaledger/wasp/packages/hashing"
	"github.com/iotaledger/wasp/packages/iscp"
	"github.com/iotaledger/wasp/packages/iscp/coreutil"
	"github.com/iotaledger/wasp/packages/metrics/nodeconnmetrics"
	"github.com/iotaledger/wasp/packages/peering"
	"github.com/iotaledger/wasp/packages/state"
	"github.com/iotaledger/wasp/packages/tcrypto"
	"github.com/iotaledger/wasp/packages/util/ready"
	"github.com/iotaledger/wasp/packages/vm/core/governance"
	"github.com/iotaledger/wasp/packages/vm/processors"
)

type ChainCore interface {
	ID() *iscp.ChainID
	GetCommitteeInfo() *CommitteeInfo
	StateCandidateToStateManager(state.VirtualStateAccess, ledgerstate.OutputID)
	TriggerChainTransition(*ChainTransitionEventData)
	Processors() *processors.Cache
	GlobalStateSync() coreutil.ChainStateSync
	GetStateReader() state.OptimisticStateReader
	GetChainNodes() []peering.PeerStatusProvider     // CommitteeNodes + AccessNodes
	GetCandidateNodes() []*governance.AccessNodeInfo // All the current candidates.
	Log() *logger.Logger

	// Most of these methods are made public for mocking in tests
	EnqueueDismissChain(reason string) // This one should really be public
	EnqueueLedgerState(chainOutput *ledgerstate.AliasOutput, timestamp time.Time)
	EnqueueOffLedgerRequestMsg(msg *messages.OffLedgerRequestMsgIn)
	EnqueueRequestAckMsg(msg *messages.RequestAckMsgIn)
	EnqueueMissingRequestIDsMsg(msg *messages.MissingRequestIDsMsgIn)
	EnqueueMissingRequestMsg(msg *messages.MissingRequestMsg)
	EnqueueTimerTick(tick int)
}

// ChainEntry interface to access chain from the chain registry side
type ChainEntry interface {
	ReceiveTransaction(*ledgerstate.Transaction)
	ReceiveState(stateOutput *ledgerstate.AliasOutput, timestamp time.Time)
	Dismiss(reason string)
	IsDismissed() bool
}

// ChainRequests is an interface to query status of the request
type ChainRequests interface {
	GetRequestProcessingStatus(id iscp.RequestID) RequestProcessingStatus
	AttachToRequestProcessed(func(iscp.RequestID)) (attachID *events.Closure)
	DetachFromRequestProcessed(attachID *events.Closure)
}

type ChainMetrics interface {
	GetNodeConnectionMetrics() nodeconnmetrics.NodeConnectionMessagesMetrics
	GetConsensusWorkflowStatus() ConsensusWorkflowStatus
	GetConsensusPipeMetrics() ConsensusPipeMetrics
}

type Chain interface {
	ChainCore
	ChainRequests
	ChainEntry
	ChainMetrics
}

// Committee is ordered (indexed 0..size-1) list of peers which run the consensus
type Committee interface {
	Address() ledgerstate.Address
	Size() uint16
	Quorum() uint16
	OwnPeerIndex() uint16
	DKShare() *tcrypto.DKShare
	IsAlivePeer(peerIndex uint16) bool
	QuorumIsAlive(quorum ...uint16) bool
	PeerStatus() []*PeerStatus
	IsReady() bool
	Close()
	RunACSConsensus(value []byte, sessionID uint64, stateIndex uint32, callback func(sessionID uint64, acs [][]byte))
	GetRandomValidators(upToN int) []*ed25519.PublicKey // TODO: Remove after OffLedgerRequest dissemination is changed.
}

type (
	NodeConnectionHandleTransactionFun        func(*ledgerstate.Transaction)
	NodeConnectionHandleInclusionStateFun     func(ledgerstate.TransactionID, ledgerstate.InclusionState)
	NodeConnectionHandleOutputFun             func(ledgerstate.Output)
	NodeConnectionHandleUnspentAliasOutputFun func(*ledgerstate.AliasOutput, time.Time)
)

type NodeConnection interface {
	Subscribe(addr ledgerstate.Address)
	Unsubscribe(addr ledgerstate.Address)
	AttachToTransactionReceived(*ledgerstate.AliasAddress, NodeConnectionHandleTransactionFun)
	AttachToInclusionStateReceived(*ledgerstate.AliasAddress, NodeConnectionHandleInclusionStateFun)
	AttachToOutputReceived(*ledgerstate.AliasAddress, NodeConnectionHandleOutputFun)
	AttachToUnspentAliasOutputReceived(*ledgerstate.AliasAddress, NodeConnectionHandleUnspentAliasOutputFun)
	PullState(addr *ledgerstate.AliasAddress)
	PullTransactionInclusionState(addr ledgerstate.Address, txid ledgerstate.TransactionID)
	PullConfirmedOutput(addr ledgerstate.Address, outputID ledgerstate.OutputID)
	PostTransaction(tx *ledgerstate.Transaction)
	GetMetrics() nodeconnmetrics.NodeConnectionMetrics
	DetachFromTransactionReceived(*ledgerstate.AliasAddress)
	DetachFromInclusionStateReceived(*ledgerstate.AliasAddress)
	DetachFromOutputReceived(*ledgerstate.AliasAddress)
	DetachFromUnspentAliasOutputReceived(*ledgerstate.AliasAddress)
	Close()
}

type ChainNodeConnection interface {
	AttachToTransactionReceived(NodeConnectionHandleTransactionFun)
	AttachToInclusionStateReceived(NodeConnectionHandleInclusionStateFun)
	AttachToOutputReceived(NodeConnectionHandleOutputFun)
	AttachToUnspentAliasOutputReceived(NodeConnectionHandleUnspentAliasOutputFun)
	PullState()
	PullTransactionInclusionState(txid ledgerstate.TransactionID)
	PullConfirmedOutput(outputID ledgerstate.OutputID)
	PostTransaction(tx *ledgerstate.Transaction)
	GetMetrics() nodeconnmetrics.NodeConnectionMessagesMetrics
	DetachFromTransactionReceived()
	DetachFromInclusionStateReceived()
	DetachFromOutputReceived()
	DetachFromUnspentAliasOutputReceived()
	Close()
}

type StateManager interface {
	Ready() *ready.Ready
	EnqueueGetBlockMsg(msg *messages.GetBlockMsgIn)
	EnqueueBlockMsg(msg *messages.BlockMsgIn)
	EnqueueStateMsg(msg *messages.StateMsg)
	EnqueueOutputMsg(msg ledgerstate.Output)
	EnqueueStateCandidateMsg(state.VirtualStateAccess, ledgerstate.OutputID)
	EnqueueTimerMsg(msg messages.TimerTick)
	GetStatusSnapshot() *SyncInfo
	SetChainPeers(peers []*ed25519.PublicKey)
	Close()
}

type Consensus interface {
	EnqueueStateTransitionMsg(state.VirtualStateAccess, *ledgerstate.AliasOutput, time.Time)
	EnqueueSignedResultMsg(*messages.SignedResultMsgIn)
	EnqueueSignedResultAckMsg(*messages.SignedResultAckMsgIn)
	EnqueueInclusionsStateMsg(ledgerstate.TransactionID, ledgerstate.InclusionState)
	EnqueueAsynchronousCommonSubsetMsg(msg *messages.AsynchronousCommonSubsetMsg)
	EnqueueVMResultMsg(msg *messages.VMResultMsg)
	EnqueueTimerMsg(messages.TimerTick)
	IsReady() bool
	Close()
	GetStatusSnapshot() *ConsensusInfo
	GetWorkflowStatus() ConsensusWorkflowStatus
	ShouldReceiveMissingRequest(req iscp.Request) bool
	GetPipeMetrics() ConsensusPipeMetrics
}

type Mempool interface {
	ReceiveRequests(reqs ...iscp.Request)
	ReceiveRequest(req iscp.Request) bool
	RemoveRequests(reqs ...iscp.RequestID)
	ReadyNow(nowis ...time.Time) []iscp.Request
	ReadyFromIDs(nowis time.Time, reqIDs ...iscp.RequestID) ([]iscp.Request, []int, bool)
	HasRequest(id iscp.RequestID) bool
	GetRequest(id iscp.RequestID) iscp.Request
	Info() MempoolInfo
	WaitRequestInPool(reqid iscp.RequestID, timeout ...time.Duration) bool // for testing
	WaitInBufferEmpty(timeout ...time.Duration) bool                       // for testing
	Close()
}

type AsynchronousCommonSubsetRunner interface {
	RunACSConsensus(value []byte, sessionID uint64, stateIndex uint32, callback func(sessionID uint64, acs [][]byte))
	Close()
}

type WAL interface {
	Write(bytes []byte) error
	Contains(i uint32) bool
	Read(i uint32) ([]byte, error)
}

type MempoolInfo struct {
	TotalPool      int
	ReadyCounter   int
	InBufCounter   int
	OutBufCounter  int
	InPoolCounter  int
	OutPoolCounter int
}

type SyncInfo struct {
	Synced                bool
	SyncedBlockIndex      uint32
	SyncedStateHash       hashing.HashValue
	SyncedStateTimestamp  time.Time
	StateOutputBlockIndex uint32
	StateOutputID         ledgerstate.OutputID
	StateOutputHash       hashing.HashValue
	StateOutputTimestamp  time.Time
}

type ConsensusInfo struct {
	StateIndex uint32
	Mempool    MempoolInfo
	TimerTick  int
}

type ConsensusWorkflowStatus interface {
	IsStateReceived() bool
	IsBatchProposalSent() bool
	IsConsensusBatchKnown() bool
	IsVMStarted() bool
	IsVMResultSigned() bool
	IsTransactionFinalized() bool
	IsTransactionPosted() bool
	IsTransactionSeen() bool
	IsInProgress() bool
	GetBatchProposalSentTime() time.Time
	GetConsensusBatchKnownTime() time.Time
	GetVMStartedTime() time.Time
	GetVMResultSignedTime() time.Time
	GetTransactionFinalizedTime() time.Time
	GetTransactionPostedTime() time.Time
	GetTransactionSeenTime() time.Time
	GetCompletedTime() time.Time
	GetCurrentStateIndex() uint32
}

type ConsensusPipeMetrics interface {
	GetEventStateTransitionMsgPipeSize() int
	GetEventSignedResultMsgPipeSize() int
	GetEventSignedResultAckMsgPipeSize() int
	GetEventInclusionStateMsgPipeSize() int
	GetEventACSMsgPipeSize() int
	GetEventVMResultMsgPipeSize() int
	GetEventTimerMsgPipeSize() int
}

type ReadyListRecord struct {
	Request iscp.Request
	Seen    map[uint16]bool
}

type CommitteeInfo struct {
	Address       ledgerstate.Address
	Size          uint16
	Quorum        uint16
	QuorumIsAlive bool
	PeerStatus    []*PeerStatus
}

type PeerStatus struct {
	Index     int
	PubKey    *ed25519.PublicKey
	NetID     string
	Connected bool
}

type ChainTransitionEventData struct {
	VirtualState    state.VirtualStateAccess
	ChainOutput     *ledgerstate.AliasOutput
	OutputTimestamp time.Time
}

func (p *PeerStatus) String() string {
	return fmt.Sprintf("%+v", *p)
}

type RequestProcessingStatus int

const (
	RequestProcessingStatusUnknown = RequestProcessingStatus(iota)
	RequestProcessingStatusBacklog
	RequestProcessingStatusCompleted
)

const (
	// TimerTickPeriod time tick for consensus and state manager objects
	TimerTickPeriod = 100 * time.Millisecond
)

const (
	PeerMsgTypeMissingRequestIDs = iota
	PeerMsgTypeMissingRequest
	PeerMsgTypeOffLedgerRequest
	PeerMsgTypeRequestAck
)
