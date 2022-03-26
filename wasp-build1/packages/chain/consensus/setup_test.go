// Copyright 2020 IOTA Stiftung
// SPDX-License-Identifier: Apache-2.0

package consensus

import (
	"fmt"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/iotaledger/goshimmer/packages/ledgerstate"
	"github.com/iotaledger/goshimmer/packages/ledgerstate/utxodb"
	"github.com/iotaledger/goshimmer/packages/ledgerstate/utxoutil"
	"github.com/iotaledger/hive.go/crypto/ed25519"
	"github.com/iotaledger/hive.go/kvstore"
	"github.com/iotaledger/hive.go/kvstore/mapdb"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/wasp/packages/chain"
	"github.com/iotaledger/wasp/packages/chain/committee"
	"github.com/iotaledger/wasp/packages/chain/mempool"
	"github.com/iotaledger/wasp/packages/chain/messages"
	"github.com/iotaledger/wasp/packages/hashing"
	"github.com/iotaledger/wasp/packages/iscp"
	"github.com/iotaledger/wasp/packages/iscp/colored"
	"github.com/iotaledger/wasp/packages/iscp/coreutil"
	"github.com/iotaledger/wasp/packages/iscp/request"
	"github.com/iotaledger/wasp/packages/kv"
	"github.com/iotaledger/wasp/packages/metrics"
	"github.com/iotaledger/wasp/packages/peering"
	"github.com/iotaledger/wasp/packages/registry"
	"github.com/iotaledger/wasp/packages/solo"
	"github.com/iotaledger/wasp/packages/state"
	"github.com/iotaledger/wasp/packages/tcrypto"
	"github.com/iotaledger/wasp/packages/testutil"
	"github.com/iotaledger/wasp/packages/testutil/testchain"
	"github.com/iotaledger/wasp/packages/testutil/testlogger"
	"github.com/iotaledger/wasp/packages/testutil/testpeers"
	"github.com/iotaledger/wasp/packages/transaction"
	"github.com/iotaledger/wasp/packages/util"
	"github.com/iotaledger/wasp/packages/wal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

type MockedEnv struct {
	T                 *testing.T
	Quorum            uint16
	Log               *logger.Logger
	Ledger            *utxodb.UtxoDB
	StateAddress      ledgerstate.Address
	OriginatorKeyPair *ed25519.KeyPair
	OriginatorAddress ledgerstate.Address
	NodeIDs           []string
	NetworkProviders  []peering.NetworkProvider
	NetworkBehaviour  *testutil.PeeringNetDynamic
	NetworkCloser     io.Closer
	DKSRegistries     []registry.DKShareRegistryProvider
	ChainID           *iscp.ChainID
	MockedACS         chain.AsynchronousCommonSubsetRunner
	InitStateOutput   *ledgerstate.AliasOutput
	mutex             sync.Mutex
	Nodes             []*mockedNode
}

type mockedNode struct {
	NodeID      string
	Env         *MockedEnv
	NodeConn    *testchain.MockedNodeConn  // GoShimmer mock
	ChainCore   *testchain.MockedChainCore // Chain mock
	stateSync   coreutil.ChainStateSync    // Chain mock
	Mempool     chain.Mempool              // Consensus needs
	Consensus   chain.Consensus            // Consensus needs
	store       kvstore.KVStore            // State manager mock
	SolidState  state.VirtualStateAccess   // State manager mock
	StateOutput *ledgerstate.AliasOutput   // State manager mock
	Log         *logger.Logger
	mutex       sync.Mutex
}

func NewMockedEnv(t *testing.T, n, quorum uint16, debug bool) (*MockedEnv, *ledgerstate.Transaction) {
	return newMockedEnv(t, n, quorum, debug, false)
}

func NewMockedEnvWithMockedACS(t *testing.T, n, quorum uint16, debug bool) (*MockedEnv, *ledgerstate.Transaction) {
	return newMockedEnv(t, n, quorum, debug, true)
}

func newMockedEnv(t *testing.T, n, quorum uint16, debug, mockACS bool) (*MockedEnv, *ledgerstate.Transaction) {
	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}
	log := testlogger.WithLevel(testlogger.NewLogger(t, "04:05.000"), level, false)
	var err error

	log.Infof("creating test environment with N = %d, T = %d", n, quorum)

	ret := &MockedEnv{
		T:      t,
		Quorum: quorum,
		Log:    log,
		Ledger: utxodb.New(),
		Nodes:  make([]*mockedNode, n),
	}
	if mockACS {
		ret.MockedACS = testchain.NewMockedACSRunner(quorum, log)
		log.Infof("running MOCKED ACS consensus")
	} else {
		log.Infof("running REAL ACS consensus")
	}

	ret.NetworkBehaviour = testutil.NewPeeringNetDynamic(log)

	log.Infof("running DKG and setting up mocked network..")
	nodeIDs, nodeIdentities := testpeers.SetupKeys(n)
	ret.NodeIDs = nodeIDs
	ret.StateAddress, ret.DKSRegistries = testpeers.SetupDkgPregenerated(t, quorum, nodeIdentities, tcrypto.DefaultSuite())
	ret.NetworkProviders, ret.NetworkCloser = testpeers.SetupNet(ret.NodeIDs, nodeIdentities, ret.NetworkBehaviour, log)

	ret.OriginatorKeyPair, ret.OriginatorAddress = ret.Ledger.NewKeyPairByIndex(0)
	_, err = ret.Ledger.RequestFunds(ret.OriginatorAddress)
	require.NoError(t, err)

	outputs := ret.Ledger.GetAddressOutputs(ret.OriginatorAddress)
	require.True(t, len(outputs) == 1)

	bals := colored.ToL1Map(colored.NewBalancesForIotas(100))

	txBuilder := utxoutil.NewBuilder(outputs...)
	err = txBuilder.AddNewAliasMint(bals, ret.StateAddress, state.OriginStateHash().Bytes())
	require.NoError(t, err)
	err = txBuilder.AddRemainderOutputIfNeeded(ret.OriginatorAddress, nil)
	require.NoError(t, err)
	originTx, err := txBuilder.BuildWithED25519(ret.OriginatorKeyPair)
	require.NoError(t, err)
	err = ret.Ledger.AddTransaction(originTx)
	require.NoError(t, err)

	ret.InitStateOutput, err = utxoutil.GetSingleChainedAliasOutput(originTx)
	require.NoError(t, err)

	ret.ChainID = iscp.NewChainID(ret.InitStateOutput.GetAliasAddress())

	return ret, originTx
}

func (env *MockedEnv) CreateNodes(timers ConsensusTimers) {
	for i := range env.Nodes {
		env.Nodes[i] = env.NewNode(uint16(i), timers)
	}
}

func (env *MockedEnv) NewNode(nodeIndex uint16, timers ConsensusTimers) *mockedNode { //nolint:revive
	nodeID := env.NodeIDs[nodeIndex]
	log := env.Log.Named(nodeID)
	ret := &mockedNode{
		NodeID:    nodeID,
		Env:       env,
		NodeConn:  testchain.NewMockedNodeConnection("Node_" + nodeID),
		store:     mapdb.NewMapDB(),
		ChainCore: testchain.NewMockedChainCore(env.T, env.ChainID, log),
		stateSync: coreutil.NewChainStateSync(),
		Log:       log,
	}
	ret.ChainCore.OnGlobalStateSync(func() coreutil.ChainStateSync {
		return ret.stateSync
	})
	ret.ChainCore.OnGetStateReader(func() state.OptimisticStateReader {
		return state.NewOptimisticStateReader(ret.store, ret.stateSync)
	})
	ret.NodeConn.OnPostTransaction(func(tx *ledgerstate.Transaction) {
		env.mutex.Lock()
		defer env.mutex.Unlock()

		if _, already := env.Ledger.GetTransaction(tx.ID()); !already {
			if err := env.Ledger.AddTransaction(tx); err != nil {
				ret.Log.Error(err)
				return
			}
			stateOutput := transaction.GetAliasOutput(tx, env.ChainID.AsAddress())
			require.NotNil(env.T, stateOutput)

			ret.Log.Infof("stored transaction to the ledger: %s", tx.ID().Base58())
			for _, node := range env.Nodes {
				go func(n *mockedNode) {
					n.mutex.Lock()
					defer n.mutex.Unlock()
					n.StateOutput = stateOutput
					n.checkStateApproval()
				}(node)
			}
		} else {
			ret.Log.Infof("transaction already in the ledger: %s", tx.ID().Base58())
		}
	})
	ret.NodeConn.OnPullTransactionInclusionState(func(txid ledgerstate.TransactionID) {
		if _, already := env.Ledger.GetTransaction(txid); already {
			go ret.Consensus.EnqueueInclusionsStateMsg(txid, ledgerstate.Confirmed)
		}
	})
	mempoolMetrics := metrics.DefaultChainMetrics()
	ret.Mempool = mempool.New(ret.ChainCore.GetStateReader(), iscp.NewInMemoryBlobCache(), log, mempoolMetrics)

	//
	// Pass the ACS mock, if it was set in env.MockedACS.
	acs := make([]chain.AsynchronousCommonSubsetRunner, 0, 1)
	if env.MockedACS != nil {
		acs = append(acs, env.MockedACS)
	}
	dkShare, err := env.DKSRegistries[nodeIndex].LoadDKShare(env.StateAddress)
	if err != nil {
		panic(err)
	}
	cmt, cmtPeerGroup, err := committee.New(
		dkShare,
		env.ChainID,
		env.NetworkProviders[nodeIndex],
		log,
		acs...,
	)
	require.NoError(env.T, err)
	cmtPeerGroup.Attach(peering.PeerMessageReceiverConsensus, func(peerMsg *peering.PeerMessageGroupIn) {
		log.Debugf("Consensus received peer message from %v of type %v", peerMsg.SenderPubKey.String(), peerMsg.MsgType)
		switch peerMsg.MsgType {
		case peerMsgTypeSignedResult:
			msg, err := messages.NewSignedResultMsg(peerMsg.MsgData)
			if err != nil {
				log.Error(err)
				return
			}
			ret.Consensus.EnqueueSignedResultMsg(&messages.SignedResultMsgIn{
				SignedResultMsg: *msg,
				SenderIndex:     peerMsg.SenderIndex,
			})
		case peerMsgTypeSignedResultAck:
			msg, err := messages.NewSignedResultAckMsg(peerMsg.MsgData)
			if err != nil {
				log.Error(err)
				return
			}
			ret.Consensus.EnqueueSignedResultAckMsg(&messages.SignedResultAckMsgIn{
				SignedResultAckMsg: *msg,
				SenderIndex:        peerMsg.SenderIndex,
			})
		}
	})

	ret.StateOutput = env.InitStateOutput
	ret.SolidState, err = state.CreateOriginState(ret.store, env.ChainID)
	ret.stateSync.SetSolidIndex(0)
	require.NoError(env.T, err)

	cons := New(ret.ChainCore, ret.Mempool, cmt, cmtPeerGroup, ret.NodeConn, true, metrics.DefaultChainMetrics(), wal.NewDefault(), timers)
	cons.(*consensus).vmRunner = testchain.NewMockedVMRunner(env.T, log)
	ret.Consensus = cons

	ret.ChainCore.OnStateCandidate(func(newState state.VirtualStateAccess, approvingOutputID ledgerstate.OutputID) {
		go func() {
			ret.mutex.Lock()
			defer ret.mutex.Unlock()
			ret.Log.Infof("chainCore.StateCandidateMsg: state hash: %s, approving output: %s",
				newState.StateCommitment(), iscp.OID(approvingOutputID))

			if ret.SolidState != nil && ret.SolidState.BlockIndex() == newState.BlockIndex() {
				ret.Log.Debugf("new state already committed for index %d", newState.BlockIndex())
				return
			}
			err := newState.Commit()
			require.NoError(env.T, err)

			ret.SolidState = newState
			ret.Log.Debugf("committed new state for index %d", newState.BlockIndex())

			ret.checkStateApproval()
		}()
	})
	return ret
}

func (env *MockedEnv) nodeCount() int {
	return len(env.NodeIDs)
}

func (env *MockedEnv) SetInitialConsensusState() {
	env.mutex.Lock()
	defer env.mutex.Unlock()

	for _, node := range env.Nodes {
		go func(n *mockedNode) {
			if n.SolidState != nil && n.SolidState.BlockIndex() == 0 {
				n.EventStateTransition()
			}
		}(node)
	}
}

func (n *mockedNode) checkStateApproval() {
	if n.SolidState == nil || n.StateOutput == nil {
		return
	}
	if n.SolidState.BlockIndex() != n.StateOutput.GetStateIndex() {
		return
	}
	stateHash, err := hashing.HashValueFromBytes(n.StateOutput.GetStateData())
	require.NoError(n.Env.T, err)
	require.EqualValues(n.Env.T, stateHash, n.SolidState.StateCommitment())

	reqIDsForLastState := make([]iscp.RequestID, 0)
	prefix := kv.Key(util.Uint32To4Bytes(n.SolidState.BlockIndex()))
	err = n.SolidState.KVStoreReader().Iterate(prefix, func(key kv.Key, value []byte) bool {
		reqid, err := iscp.RequestIDFromBytes(value)
		require.NoError(n.Env.T, err)
		reqIDsForLastState = append(reqIDsForLastState, reqid)
		return true
	})
	require.NoError(n.Env.T, err)
	n.Mempool.RemoveRequests(reqIDsForLastState...)

	n.Log.Infof("STATE APPROVED (%d reqs). Index: %d, State output: %s",
		len(reqIDsForLastState), n.SolidState.BlockIndex(), iscp.OID(n.StateOutput.ID()))

	n.EventStateTransition()
}

func (n *mockedNode) EventStateTransition() {
	n.Log.Debugf("EventStateTransition")

	n.ChainCore.GlobalStateSync().SetSolidIndex(n.SolidState.BlockIndex())

	n.Consensus.EnqueueStateTransitionMsg(n.SolidState.Copy(), n.StateOutput, time.Now())
}

func (env *MockedEnv) StartTimers() {
	for _, n := range env.Nodes {
		n.StartTimer()
	}
}

func (n *mockedNode) StartTimer() {
	n.Log.Debugf("started timer..")
	go func() {
		counter := 0
		for {
			n.Consensus.EnqueueTimerMsg(messages.TimerTick(counter))
			counter++
			time.Sleep(50 * time.Millisecond)
		}
	}()
}

func (env *MockedEnv) WaitTimerTick(until int) error {
	checkTimerTickFun := func(node *mockedNode) bool {
		snap := node.Consensus.GetStatusSnapshot()
		if snap != nil && snap.TimerTick >= until {
			return true
		}
		return false
	}
	return env.WaitForEventFromNodes("TimerTick", checkTimerTickFun)
}

func (env *MockedEnv) WaitStateIndex(quorum int, stateIndex uint32, timeout ...time.Duration) error {
	checkStateIndexFun := func(node *mockedNode) bool {
		snap := node.Consensus.GetStatusSnapshot()
		if snap != nil && snap.StateIndex >= stateIndex {
			return true
		}
		return false
	}
	return env.WaitForEventFromNodesQuorum("stateIndex", quorum, checkStateIndexFun, timeout...)
}

func (env *MockedEnv) WaitMempool(numRequests int, quorum int, timeout ...time.Duration) error { //nolint:gocritic
	checkMempoolFun := func(node *mockedNode) bool {
		snap := node.Consensus.GetStatusSnapshot()
		if snap != nil && snap.Mempool.InPoolCounter >= numRequests && snap.Mempool.OutPoolCounter >= numRequests {
			return true
		}
		return false
	}
	return env.WaitForEventFromNodesQuorum("mempool", quorum, checkMempoolFun, timeout...)
}

func (env *MockedEnv) WaitForEventFromNodes(waitName string, nodeConditionFun func(node *mockedNode) bool, timeout ...time.Duration) error {
	return env.WaitForEventFromNodesQuorum(waitName, env.nodeCount(), nodeConditionFun, timeout...)
}

func (env *MockedEnv) WaitForEventFromNodesQuorum(waitName string, quorum int, isEventOccuredFun func(node *mockedNode) bool, timeout ...time.Duration) error {
	to := 10 * time.Second
	if len(timeout) > 0 {
		to = timeout[0]
	}
	ch := make(chan int)
	nodeCount := env.nodeCount()
	deadline := time.Now().Add(to)
	for _, n := range env.Nodes {
		go func(node *mockedNode) {
			for time.Now().Before(deadline) {
				if isEventOccuredFun(node) {
					ch <- 1
				}
				time.Sleep(10 * time.Millisecond)
			}
			ch <- 0
		}(n)
	}
	var sum, total int
	for n := range ch {
		sum += n
		total++
		if sum >= quorum {
			return nil
		}
		if total >= nodeCount {
			return fmt.Errorf("Wait for %s: test timed out", waitName)
		}
	}
	return fmt.Errorf("WaitMempool: timeout expired %v", to)
}

func (env *MockedEnv) PostDummyRequests(n int, randomize ...bool) {
	reqs := make([]*request.OffLedger, n)
	for i := 0; i < n; i++ {
		reqs[i] = solo.NewCallParams("dummy", "dummy", "c", i).
			NewRequestOffLedger(iscp.RandomChainID(), env.OriginatorKeyPair)
	}
	rnd := len(randomize) > 0 && randomize[0]
	for _, n := range env.Nodes {
		for _, req := range reqs {
			go func(node *mockedNode, r *request.OffLedger) {
				if rnd {
					time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
				}
				node.Mempool.ReceiveRequest(r)
			}(n, req)
		}
	}
}
