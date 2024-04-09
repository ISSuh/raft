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
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/log"
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

const (
	BackgroundLoopLen = 2
)

type RaftNode struct {
	*Node

	metadata        *message.NodeMetadata
	nodeEventChan   chan event.Event
	peerNodeManager *PeerNodeManager
	worker          map[State]Worker

	storage storage.Engine
	logs    *log.Logs

	quit chan struct{}
}

func NewRaftNode(
	metadata *message.NodeMetadata, nodeEventChan chan event.Event, peerNodeManager *PeerNodeManager, quit chan struct{},
) *RaftNode {
	node := NewNode(metadata)
	logs := log.NewLogs()

	n := &RaftNode{
		Node: node,

		metadata:        metadata,
		nodeEventChan:   nodeEventChan,
		peerNodeManager: peerNodeManager,
		worker:          make(map[State]Worker),

		storage: storage.NewMapEngine(),
		logs:    logs,

		quit: quit,
	}

	n.worker[FollowerState] = NewFollowerStateWorker(node, n, n.quit)
	n.worker[CandidateState] = NewCandidateStateWorker(node, peerNodeManager, n, n.quit)
	n.worker[LeaderState] = NewLeaderStateWorker(node, logs, peerNodeManager, n, n.quit)
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
	go n.peerNodeManager.clusterEventLoop(c)
	go n.nodeStateLoop(c)
}

func (n *RaftNode) Stop() {
	go func() {
		n.setState(StopState)

		for i := 0; i < BackgroundLoopLen; i++ {
			n.quit <- struct{}{}
		}
	}()
}

func (n *RaftNode) Submit(command []byte) error {
	state := n.currentState()
	if state != LeaderState {
		return fmt.Errorf("[RaftNode.Submit] currunt state is not leader. state : %s", state)
	}

	l := []*message.LogEntry{
		{Term: n.currentTerm(), Log: command},
	}

	n.logs.AppendLog(l)
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
			n.worker[state].Work(c)
			state = n.currentState()
		}
	}
}

func (n *RaftNode) WaitUntilEmit() <-chan event.Event {
	return n.nodeEventChan
}

func (n *RaftNode) ProcessEvent(e event.Event) (interface{}, error) {
	var result interface{}
	var err error
	switch e.Type {
	case event.ReqeustVote:
		result, err = n.onRequestVote(e)
	case event.AppendEntries:
		result, err = n.onAppendEntries(e)
	default:
		result = nil
		err = fmt.Errorf("[ProcessEvent] invalid event type. type : %s", e.Type.String())
	}
	return result, err
}

func (n *RaftNode) onRequestVote(e event.Event) (*message.RequestVoteReply, error) {
	logger.Debug("[onRequestVote]")
	requestVoteMessage, ok := e.Message.(*message.RequestVote)
	if !ok {
		return nil, fmt.Errorf("[onRequestVote] can not convert to *message.RequestVoteReply. %v", e)
	}

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

	// peer term is less than me. return false
	if msg.Term < n.currentTerm() {
		r := &message.AppendEntriesReply{
			Term:    n.currentTerm(),
			Success: false,
			PeerId:  n.meta.Id,
		}
		return r, nil
	}

	logger.Debug("[onAppendEntries] entry : %v / len : %d", *msg, n.logs.Len())

	// peer term is bigger than me.
	n.setState(FollowerState)
	n.setTerm(msg.Term)

	// check validate between recieved prev log entry and last log entry
	isValid, err := n.isValidPrevLogIndexAndTerm(msg.PrevLogTerm, msg.PrevLogIndex)
	if err != nil {
		return nil, err
	}

	if !isValid {
		lastIndex := int64(n.logs.Len() - 1)
		conflictIndex := util.Min(lastIndex, msg.PrevLogIndex)

		conflictTerm := uint64(0)
		if conflictIndex > 0 {
			conflictTerm, err = n.logs.EntryTerm(conflictIndex)
			if err != nil {
				return nil, err
			}
		}

		r := &message.AppendEntriesReply{
			Term:          n.currentTerm(),
			Success:       false,
			PeerId:        n.meta.Id,
			ConflictIndex: conflictIndex,
			ConflictTerm:  conflictTerm,
		}
		return r, nil
	}

	// find received entries index and node.log index for save entries
	logIndex := msg.PrevLogIndex + 1
	logLen := n.logs.Len()
	newLogIndex := 0
	newLogLen := len(msg.Entries)
	applyEntriesLen := int64(0)
	for {
		if (logIndex >= int64(logLen)) || (newLogIndex >= newLogLen) {
			break
		}

		term, err := n.logs.EntryTerm(logIndex)
		if err != nil {
			return nil, err
		}

		if term != msg.Entries[newLogIndex].Term {
			break
		}

		logIndex++
		newLogIndex++
	}

	// update log entries
	if newLogIndex < newLogLen {
		applyEntries := msg.Entries[newLogIndex:]
		if err := n.logs.AppendLogSinceToIndex(logIndex, applyEntries); err != nil {
			return nil, err
		}
		applyEntriesLen = int64(newLogLen - newLogIndex)
	}

	// commit
	commitIndex := n.logs.CommitIndex()
	if msg.LeaderCommitIndex > commitIndex {
		updatedlogLen := n.logs.Len()
		updatedLogIndex := int64(updatedlogLen - 1)
		newCommitIndex := util.Min(msg.LeaderCommitIndex, updatedLogIndex)
		n.logs.UpdateCommitIndex(newCommitIndex)

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

func (n *RaftNode) isValidPrevLogIndexAndTerm(prevLogTerm uint64, prevLogIndex int64) (bool, error) {
	// initail index when first request
	if prevLogIndex == -1 {
		return true, nil
	}

	lastLogIndex := int64(n.logs.Len() - 1)
	if prevLogIndex > lastLogIndex {
		return false, nil
	}

	lastLogTermOnPrevIndex, err := n.logs.EntryTerm(prevLogIndex)
	if err != nil {
		return false, err
	}

	if prevLogTerm != lastLogTermOnPrevIndex {
		return false, nil
	}
	return true, nil
}

// func (node *RaftNode) saveToStorage() {
// 	node.logMutex.Lock()
// 	defer node.logMutex.Unlock()

// 	var termBuffer bytes.Buffer
// 	if err := gob.NewEncoder(&termBuffer).Encode(node.currentTerm()); err != nil {
// 		log.WithField("node", "node.saveToStorage").Fatal(goidForlog()+"err : ", err)
// 		return
// 	}

// 	node.storage.Set(StorageTermKey, termBuffer.Bytes())

// 	var logBuffer bytes.Buffer
// 	if err := gob.NewEncoder(&logBuffer).Encode(node.logs); err != nil {
// 		log.WithField("node", "node.saveToStorage").Fatal(goidForlog()+"err : ", err)
// 		return
// 	}

// 	node.storage.Set(StorageLogKey, logBuffer.Bytes())

// 	// run entry updated callback
// 	// need redesign
// 	go func() {
// 		node.logMutex.Lock()
// 		defer node.logMutex.Unlock()

// 		lastEntry := node.logs[len(node.logs)-1]
// 		node.entryHandler.EntryUpdated(lastEntry.Log)
// 	}()
// }

// func (n *RaftNode) restoreFromStorage() {
// 	if buffer := n.storage.Get(StorageTermKey); buffer != nil {
// 		data := gob.NewDecoder(bytes.NewBuffer(buffer))
// 		var term uint64 = 0
// 		if err := data.Decode(&term); err == nil {
// 			log.WithField("node", "node.restoreFromStorage").Fatal(goidForlog()+"err : ", err)
// 		} else {
// 			node.setTerm(term)
// 		}
// 	}

// 	if buffer := node.storage.Get(StorageLogKey); buffer != nil {
// 		data := gob.NewDecoder(bytes.NewBuffer(buffer))
// 		if err := data.Decode(&node.logs); err != nil {
// 			log.WithField("node", "node.restoreFromStorage").Fatal(goidForlog()+"err : ", err)
// 		}
// 	}
// }
