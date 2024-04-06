/*
MIT License

Copyright (c) 2024 ISSuh

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package node

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/logger"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/storage"
	"github.com/ISSuh/raft/internal/util"
)

const (
	DefaultElectionMinTimeout = 150 * time.Millisecond
	DefaultElectionMaxTimeout = 300 * time.Millisecond

	DefaultHeartBeatMinTimeout = 50 * time.Millisecond
	DefaultHeartBeatMaxTimeout = 150 * time.Millisecond
)

type RaftNode struct {
	*Node

	metadata         *message.NodeMetadata
	nodeEventChan    chan event.Event
	clusterEventChan chan event.Event
	peerNodeManager  *PeerNodeManager

	workGroup sync.WaitGroup
	timer     *time.Timer

	leaderId int32

	storage     storage.Engine
	entries     []*message.LogEntry
	commitIndex int64
	// map[peerId]logIndex
	nextIndex map[int32]int64
	// map[peerId]logIndex
	matchIndex map[int32]int64
	logMutex   sync.Mutex

	quit chan struct{}
}

func NewRaftNode(
	metadata *message.NodeMetadata, nodeEventChan chan event.Event, clusterEventChan chan event.Event, peerNodeManager *PeerNodeManager,
) *RaftNode {
	n := &RaftNode{
		Node: NewNode(metadata),

		metadata:         metadata,
		nodeEventChan:    nodeEventChan,
		clusterEventChan: clusterEventChan,
		peerNodeManager:  peerNodeManager,

		workGroup: sync.WaitGroup{},
		timer:     nil,

		leaderId: -1,

		storage:     storage.NewMapEngine(),
		entries:     make([]*message.LogEntry, 0),
		commitIndex: -1,
		nextIndex:   map[int32]int64{},
		matchIndex:  map[int32]int64{},

		quit: make(chan struct{}, 2),
	}
	return n
}

func (n *RaftNode) NodeMetaData() *message.NodeMetadata {
	return n.metadata
}

func (n *RaftNode) ConnectToPeerNode(peerNodes *message.NodeMetadataesList) error {
	for _, peerNode := range peerNodes.Nodes {
		err := n.peerNodeManager.registPeerNode(peerNode)
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *RaftNode) Run(c context.Context) {
	go n.clusterEventLoop(c)
	go n.nodeStateLoop(c)
}

func (n *RaftNode) Stop() {
	go func() {
		n.setState(StopState)
		n.quit <- struct{}{}
	}()
}

func (n *RaftNode) Submit(command []byte) error {
	state := n.currentState()
	if state != LeaderState {
		return fmt.Errorf("[RaftNode.Submit] currunt state is not leader. state : %s", state)
	}

	n.logMutex.Lock()
	defer n.logMutex.Unlock()

	n.entries = append(n.entries, &message.LogEntry{
		Term: n.currentTerm(),
		Log:  command,
	})
	return nil
}

func (n *RaftNode) nodeStateLoop(c context.Context) {
	state := n.currentState()
	for state != StopState {
		logger.Debug("[nodeStateLoop]")
		select {
		case <-c.Done():
			logger.Info("[nodeStateLoop] context done")
			return
		case <-n.quit:
			logger.Info("[nodeStateLoop] force quit")
			return
		default:
			switch state {
			case FollowerState:
				n.followerStateLoop(c)
			case CandidateState:
				n.candidateStateLoop(c)
			case LeaderState:
				n.leaderStateLoop(c)
			}

			state = n.currentState()
		}
	}
}

func (n *RaftNode) clusterEventLoop(c context.Context) {
	for {
		select {
		case <-c.Done():
			logger.Info("[clusterEventLoop] context done")
		case <-n.quit:
			logger.Info("[clusterEventLoop] force quit")
		case e := <-n.clusterEventChan:
			result, err := n.processClusterEvent(e)
			if err != nil {
				logger.Info("[clusterEventLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (n *RaftNode) processClusterEvent(e event.Event) (interface{}, error) {
	var result interface{}
	var err error
	switch e.Type {
	case event.NotifyNodeConnected:
		result, err = n.onNotifyNodeConnected(e)
	case event.NotifyNodeDisconnected:
		result, err = n.onNotifyNodeDisconnected(e)
	default:
		result = nil
		err = fmt.Errorf("[processClusterEvent] invalid event type. type : %s", e.Type.String())
	}
	return result, err
}

func (n *RaftNode) onNotifyNodeConnected(e event.Event) (bool, error) {
	logger.Debug("[onNotifyNodeConnected]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return false, fmt.Errorf("[onNotifyNodeConnected] can not convert to *message.NodeMetadata. %v", e)
	}

	err := n.peerNodeManager.registPeerNode(node)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (n *RaftNode) onNotifyNodeDisconnected(e event.Event) (bool, error) {
	logger.Debug("[onNotifyNodeDisconnected]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return false, fmt.Errorf("[onNotifyNodeDisconnected] can not convert to *message.NodeMetadata. %v", e)
	}

	logger.Info("[onNotifyNodeDisconnected] remove peer node.  %+v", node)
	n.peerNodeManager.removePeerNode(node.Id)
	return true, nil
}

func (n *RaftNode) followerStateLoop(c context.Context) {
	logger.Debug("[followerStateLoop]")
	for n.currentState() == FollowerState {
		timeout := util.Timout(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)
		n.timer = time.NewTimer(timeout)

		select {
		case <-c.Done():
			logger.Info("[followerStateLoop] context done\n")
			n.setState(StopState)
		case <-n.quit:
			logger.Info("[followerStateLoop] force quit\n")
			n.setState(StopState)
		case <-n.timer.C:
			n.setState(CandidateState)
		case e := <-n.nodeEventChan:
			result, err := n.processNodeEvent(e)
			if err != nil {
				logger.Info("[followerStateLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (n *RaftNode) candidateStateLoop(c context.Context) {
	logger.Debug("[candidateStateLoop]")
	var replyChan chan *message.RequestVoteReply
	electionGrantedCount := 0
	needEraction := true
	n.leaderId = -1

	for n.currentState() == CandidateState {
		if needEraction {
			n.increaseTerm()
			electionGrantedCount++
			n.leaderId = n.metadata.Id

			replyChan = n.doEraction()

			if n.timer != nil {
				n.timer.Stop()
			}

			timeout := util.Timout(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)
			n.timer = time.NewTimer(timeout)
			needEraction = false
		}

		peerNodeLen := n.peerNodeManager.numberOfPeer()
		majorityCount := 0
		if peerNodeLen == 1 {
			majorityCount = 2
		} else {
			majorityCount = (peerNodeLen / 2) + 1
		}
		logger.Info(
			"[candidateStateLoop] peer nodel len : %d, electionGrantedCount : %d, majorityCount : %d",
			peerNodeLen, electionGrantedCount, majorityCount,
		)
		if electionGrantedCount == majorityCount {
			n.setState(LeaderState)
			return
		}

		select {
		case <-c.Done():
			logger.Info("[candidateStateLoop] context done\n")
			n.setState(StopState)
		case <-n.quit:
			logger.Info("[candidateStateLoop] force quit\n")
			n.setState(StopState)
		case <-n.timer.C:
			needEraction = true
			electionGrantedCount = 0
			n.setTerm(n.currentTerm() - 1)
		case reply := <-replyChan:
			if reply.Term > n.currentTerm() {
				n.setState(FollowerState)
				return
			}

			if reply.VoteGranted && reply.Term == n.currentTerm() {
				electionGrantedCount++
			}
		case e := <-n.nodeEventChan:
			result, err := n.processNodeEvent(e)
			if err != nil {
				logger.Info("[candidateStateLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (n *RaftNode) doEraction() chan *message.RequestVoteReply {
	logger.Debug("[doEraction]")
	peersLen := n.peerNodeManager.numberOfPeer()
	replyCahn := make(chan *message.RequestVoteReply, peersLen)

	requestVoteMessage := &message.RequestVote{
		Term:        n.currentTerm(),
		CandidateId: n.metadata.Id,
	}

	peerNodes := n.peerNodeManager.findAll()
	for _, peer := range peerNodes {
		n.workGroup.Add(1)
		go func(peer *RaftPeerNode, message *message.RequestVote) {
			defer n.workGroup.Done()
			if n.currentState() != CandidateState {
				return
			}

			reply, err := peer.RequestVote(message)
			if err != nil {
				logger.Info("[doEraction] %s", err.Error())
				return
			}

			replyCahn <- reply
		}(peer, requestVoteMessage)
	}
	return replyCahn
}

func (n *RaftNode) leaderStateLoop(c context.Context) {
	logger.Debug("[leaderStateLoop]")

	// var timeout <-chan time.Time
	var replyChan chan *message.AppendEntriesReply
	needHeartBeat := true
	appendSuccesCount := 0

	for n.currentState() == LeaderState {
		if needHeartBeat {
			peersLen := n.peerNodeManager.numberOfPeer()
			replyChan = make(chan *message.AppendEntriesReply, peersLen)
			n.doHeartBeat(replyChan)

			if n.timer != nil {
				n.timer.Stop()
			}

			timeout := util.Timout(DefaultHeartBeatMinTimeout, DefaultHeartBeatMaxTimeout)
			n.timer = time.NewTimer(timeout)
			needHeartBeat = false
		}

		select {
		case <-c.Done():
			logger.Info("[leaderStateLoop] context done\n")
			n.setState(StopState)
		case <-n.quit:
			logger.Info("[leaderStateLoop] force quit\n")
			n.setState(StopState)
		case <-n.timer.C:
			needHeartBeat = true
		case reply := <-replyChan:
			if !n.applyAppendEntries(reply, &appendSuccesCount) {
				n.timer.Stop()
				n.setState(FollowerState)
				close(replyChan)
				return
			}

			poerNodeNum := n.peerNodeManager.numberOfPeer()
			majorityCount := (poerNodeNum / 2) + 1
			if appendSuccesCount == majorityCount {
				logLen := len(n.entries)
				n.commitIndex = int64(logLen - 1)
			}
		case e := <-n.nodeEventChan:
			result, err := n.processNodeEvent(e)
			if err != nil {
				logger.Info("[leaderStateLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (n *RaftNode) doHeartBeat(replyChan chan *message.AppendEntriesReply) {
	logger.Debug("[doHeartBeat]")
	peerNodes := n.peerNodeManager.findAll()
	for _, peer := range peerNodes {
		n.workGroup.Add(1)
		go func(peer *RaftPeerNode, replyChan chan *message.AppendEntriesReply) {
			defer n.workGroup.Done()

			appendEntriesMessage := &message.AppendEntries{
				Term:              n.currentTerm(),
				LeaderId:          n.leaderId,
				PrevLogIndex:      -1,
				PrevLogTerm:       0,
				Entries:           make([]*message.LogEntry, 0),
				LeaderCommitIndex: n.commitIndex,
			}

			nextIndex := n.nextIndex[peer.Id()]
			prevIndex := nextIndex - 1
			appendEntriesMessage.PrevLogIndex = prevIndex
			if prevIndex >= 0 {
				appendEntriesMessage.PrevLogTerm = n.entries[prevIndex].Term
			}

			n.logMutex.Lock()
			appendEntriesMessage.Entries = n.entries[nextIndex:]
			n.logMutex.Unlock()

			reply, err := peer.AppendEntries(appendEntriesMessage)
			if err != nil {
				logger.Info("[doHeartBeat] %s", err.Error())
				return
			}

			replyChan <- reply
		}(peer, replyChan)
	}
}

func (n *RaftNode) applyAppendEntries(message *message.AppendEntriesReply, appendSuccesCount *int) bool {
	logger.Debug("[applyAppendEntries]")
	if message.Term > n.currentTerm() {
		n.setState(FollowerState)
		n.setTerm(message.Term)
		return false
	}

	peerId := message.PeerId
	if !message.Success {
		logIndex := int64(len(n.entries) - 1)
		confilctIndex := util.Min(message.ConflictIndex, logIndex)
		n.nextIndex[peerId] = util.Max(0, confilctIndex)
		return false
	}

	n.nextIndex[peerId] += message.ApplyEntriesLen
	n.matchIndex[peerId] = n.nextIndex[peerId] - 1

	*appendSuccesCount++
	return true
}

func (n *RaftNode) processNodeEvent(e event.Event) (interface{}, error) {
	var result interface{}
	var err error
	switch e.Type {
	case event.ReqeustVote:
		result, err = n.onRequestVote(e)
	case event.AppendEntries:
		result, err = n.onAppendEntries(e)
	default:
		result = nil
		err = fmt.Errorf("[processClusterEvent] invalid event type. type : %s", e.Type.String())
	}
	return result, err
}

func (n *RaftNode) onRequestVote(e event.Event) (*message.RequestVoteReply, error) {
	logger.Debug("[onRequestVote]")
	requestVoteMessage, ok := e.Message.(*message.RequestVote)
	if !ok {
		return nil, fmt.Errorf("[onRequestVote] can not convert to *message.RequestVoteReply. %v", e)
	}

	logger.Info("[onRequestVote] my term : %d, request : %s", n.currentState(), requestVoteMessage)

	// TODO : need implement case of same term of request and my term
	reply := &message.RequestVoteReply{}
	if requestVoteMessage.Term > n.currentTerm() {
		n.setTerm(requestVoteMessage.Term)
		n.setState(FollowerState)
		n.leaderId = requestVoteMessage.CandidateId

		reply.VoteGranted = true
	} else {
		reply.Term = n.currentTerm()
		reply.VoteGranted = false
	}

	return reply, nil
}

func (n *RaftNode) onAppendEntries(e event.Event) (*message.AppendEntriesReply, error) {
	logger.Info("[onAppendEntries]")
	msg, ok := e.Message.(*message.AppendEntries)
	if !ok {
		return nil, fmt.Errorf("[onAppendEntries] can not convert to *message.RequestVoteReply. %v", e)
	}

	// peer term is less than me. return false to peer
	if msg.Term < n.currentTerm() {
		r := &message.AppendEntriesReply{
			Term:    n.currentTerm(),
			Success: false,
			PeerId:  n.meta.Id,
		}
		return r, nil
	}

	// peer term is bigger than me.
	n.setState(FollowerState)
	n.setTerm(msg.Term)
	n.timer.Stop()

	// TODO : need update
	// check validate between recieved prev log entry and last log entry
	if !n.isValidPrevLogIndexAndTerm(msg.PrevLogTerm, msg.PrevLogIndex) {
		lastIndex := int64(len(n.entries) - 1)

		conflictIndex := util.Min(lastIndex, msg.PrevLogIndex)
		conflictTerm := n.entries[conflictIndex].Term

		r := &message.AppendEntriesReply{
			Term:          n.currentTerm(),
			Success:       false,
			PeerId:        n.meta.Id,
			ConflictIndex: conflictIndex,
			ConflictTerm:  conflictTerm,
		}
		return r, nil
	}

	// TODO : need update
	// find received entries index and node.log index for save entries
	logIndex := msg.PrevLogIndex + 1
	logLen := len(n.entries)
	newLogIndex := 0
	newLogLen := len(msg.Entries)
	applyEntriesLen := int64(0)
	for {
		if (logIndex >= int64(logLen)) || (newLogIndex >= newLogLen) {
			break
		}

		if n.entries[logIndex].Term != msg.Entries[newLogIndex].Term {
			break
		}

		logIndex++
		newLogIndex++
	}

	// update log entries
	if newLogIndex < newLogLen {
		n.entries = append(n.entries[:logIndex], msg.Entries[newLogIndex:]...)
		applyEntriesLen = int64(newLogLen - newLogIndex)
	}

	// commit
	if msg.LeaderCommitIndex > n.commitIndex {
		updatedlogLen := len(n.entries)
		updatedLogIndex := int64(updatedlogLen - 1)
		n.commitIndex = util.Min(msg.LeaderCommitIndex, updatedLogIndex)

		// need commit log to storage
	}

	r := &message.AppendEntriesReply{
		Term:            msg.Term,
		Success:         true,
		PeerId:          n.meta.Id,
		ApplyEntriesLen: applyEntriesLen,
	}
	return r, nil
}

func (n *RaftNode) isValidPrevLogIndexAndTerm(prevLogTerm uint64, prevLogIndex int64) bool {
	if prevLogIndex == -1 {
		return true
	}

	lastLogIndex := int64(len(n.entries) - 1)
	lastLogTermOnPrevIndex := n.entries[prevLogIndex].Term
	if (prevLogIndex < lastLogIndex) || (prevLogTerm != lastLogTermOnPrevIndex) {
		return false
	}
	return true
}
