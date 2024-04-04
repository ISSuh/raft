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
	"log"
	"sync"
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/util"
)

const (
	DefaultElectionMinTimeout = 150 * time.Millisecond
	DefaultElectionMaxTimeout = 300 * time.Millisecond

	DefaultHeartBeatMinTimeout = 100 * time.Millisecond
	DefaultHeartBeatMaxTimeout = 150 * time.Millisecond
)

type RaftNode struct {
	*NodeState

	metadata         *message.NodeMetadata
	nodeEventChan    chan event.Event
	clusterEventChan chan event.Event
	peerNodeManager  *PeerNodeManager

	leaderId  int32
	workGroup sync.WaitGroup

	quit chan struct{}
}

func NewRaftNode(
	metadata *message.NodeMetadata, nodeEventChan chan event.Event, clusterEventChan chan event.Event, peerNodeManager *PeerNodeManager,
) *RaftNode {
	return &RaftNode{
		NodeState:        NewNodeState(),
		metadata:         metadata,
		nodeEventChan:    nodeEventChan,
		clusterEventChan: clusterEventChan,
		peerNodeManager:  peerNodeManager,
		leaderId:         -1,
		quit:             make(chan struct{}, 2),
	}
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
		n.quit <- struct{}{}
	}()
}

func (n *RaftNode) nodeStateLoop(c context.Context) {
	state := n.currentState()
	for state != StopState {
		select {
		case <-c.Done():
			return
		case <-n.quit:
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
			log.Printf("[RaftNode.eventLoop] context done\n")
		case <-n.quit:
			log.Printf("[RaftNode.eventLoop] force quit\n")
		case e := <-n.clusterEventChan:
			result, err := n.processClusterEvent(e)
			if err != nil {
				log.Printf("[RaftNode.eventLoop] %s\n", err.Error())
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
		err = fmt.Errorf("[RaftNode.processClusterEvent] invalid event type. type : %s", e.Type.String())
	}
	return result, err
}

func (n *RaftNode) onNotifyNodeConnected(e event.Event) (bool, error) {
	log.Printf("[RaftNode.onNotifyNodeConnected]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return false, fmt.Errorf("[RaftNode.onNotifyNodeConnected] can not convert to *message.NodeMetadata. %v", e)
	}

	err := n.peerNodeManager.registPeerNode(node)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (n *RaftNode) onNotifyNodeDisconnected(e event.Event) (bool, error) {
	log.Printf("[RaftNode.onNotifyNodeDisconnected]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return false, fmt.Errorf("[RaftNode.onNotifyNodeDisconnected] can not convert to *message.NodeMetadata. %v", e)
	}

	log.Printf("[RaftNode.onNotifyNodeDisconnected] remove peer node.  %+v", node)
	n.peerNodeManager.removePeerNode(int(node.Id))
	return true, nil
}

func (n *RaftNode) followerStateLoop(c context.Context) {
	log.Printf("[RaftNode.followerStateLoop]")
	timeout := util.Timer(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)

	for n.currentState() == FollowerState {
		select {
		case <-c.Done():
			log.Printf("[RaftNode.followerStateLoop] context done\n")
			n.setState(StopState)
		case <-n.quit:
			log.Printf("[RaftNode.followerStateLoop] force quit\n")
			n.setState(StopState)
		case <-timeout:
			n.setState(CandidateState)
		case e := <-n.nodeEventChan:
			result, err := n.processNodeEvent(e)
			if err != nil {
				log.Printf("[RaftNode.followerStateLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}

			// TODO : need timer reset
		}
	}
}

func (n *RaftNode) candidateStateLoop(c context.Context) {
	log.Printf("[RaftNode.candidateStateLoop]")
	var replyChan chan *message.RequestVoteReply
	var timeout <-chan time.Time
	electionGrantedCount := 0
	needEraction := true
	n.leaderId = -1

	for n.currentState() == CandidateState {
		if needEraction {
			n.increaseTerm()
			electionGrantedCount++
			n.leaderId = n.metadata.Id

			replyChan = n.doEraction()

			timeout = util.Timer(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)
			needEraction = false
		}

		peerNodeLen := n.peerNodeManager.numberOfPeer()
		majorityCount := 0
		if peerNodeLen == 1 {
			majorityCount = 2
		} else {
			majorityCount = (peerNodeLen / 2) + 1
		}
		log.Printf(
			"[RaftNode.candidateStateLoop] peer nodel len : %d, electionGrantedCount : %d, majorityCount : %d",
			peerNodeLen, electionGrantedCount, majorityCount,
		)
		if electionGrantedCount == majorityCount {
			n.setState(LeaderState)
			return
		}

		select {
		case <-c.Done():
			log.Printf("[RaftNode.candidateStateLoop] context done\n")
			n.setState(StopState)
		case <-n.quit:
			log.Printf("[RaftNode.candidateStateLoop] force quit\n")
			n.setState(StopState)
		case <-timeout:
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
				log.Printf("[RaftNode.candidateStateLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (n *RaftNode) doEraction() chan *message.RequestVoteReply {
	log.Printf("[RaftNode.doEraction]")
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
				log.Printf("[RaftNode.doEraction] %s", err.Error())
				id := int(peer.metadata.Id)
				n.peerNodeManager.removePeerNode(id)
				return
			}

			replyCahn <- reply
		}(peer, requestVoteMessage)
	}
	return replyCahn
}

func (n *RaftNode) leaderStateLoop(c context.Context) {
	log.Printf("[RaftNode.leaderStateLoop]")

	var timeout <-chan time.Time
	var replyChan chan *message.AppendEntriesReply
	needHeartBeat := true
	// appendSuccesCount := 0

	for n.currentState() == LeaderState {
		if needHeartBeat {
			replyChan = n.doHeartBeat()

			timeout = util.Timer(DefaultHeartBeatMinTimeout, DefaultHeartBeatMaxTimeout)
			needHeartBeat = false
		}

		select {
		case <-c.Done():
			log.Printf("[RaftNode.leaderStateLoop] context done\n")
			n.setState(StopState)
		case <-n.quit:
			log.Printf("[RaftNode.leaderStateLoop] force quit\n")
			n.setState(StopState)
		case <-timeout:
			needHeartBeat = true
		case <-replyChan:
			// n.applyAppendEntries(reply.sendEntryLen, result.reply, &appendSuccesCount)

			// poerNodeNum := n.peerNodeManager.numberOfPeer()
			// majorityCount := (poerNodeNum / 2) + 1
			// if appendSuccesCount == majorityCount {
			// 	node.commitIndex = int64(len(node.logs) - 1)
			// }
		case e := <-n.nodeEventChan:
			result, err := n.processNodeEvent(e)
			if err != nil {
				log.Printf("[RaftNode.leaderStateLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (n *RaftNode) doHeartBeat() chan *message.AppendEntriesReply {
	log.Printf("[RaftNode.doHeartBeat]")
	peersLen := n.peerNodeManager.numberOfPeer()
	replyChan := make(chan *message.AppendEntriesReply, peersLen)

	peerNodes := n.peerNodeManager.findAll()
	for _, peer := range peerNodes {
		n.workGroup.Add(1)
		go func(peer *RaftPeerNode) {
			defer n.workGroup.Done()

			appendEntriesMessage := &message.AppendEntries{
				Term:     n.currentTerm(),
				LeaderId: int32(n.leaderId),
				// PrevLogIndex: -1,
				// PrevLogTerm:  0,
				Entries: make([]*message.LogEntry, 0),
				// LeaderCommitIndex: node.commitIndex,
			}

			// nextIndex := node.nextIndex[peer.id]
			// arg.PrevLogIndex = nextIndex - 1
			// if arg.PrevLogIndex >= 0 {
			// 	arg.PrevLogTerm = node.logs[arg.PrevLogIndex].Term
			// }

			// node.logMutex.Lock()
			// arg.Entries = node.logs[nextIndex:]
			// node.logMutex.Unlock()

			reply, err := peer.AppendEntries(appendEntriesMessage)
			if err != nil {
				log.Printf("[RaftNode.doHeartBeat] %s", err.Error())
				id := int(peer.metadata.Id)
				n.peerNodeManager.removePeerNode(id)
				return
			}

			replyChan <- reply
		}(peer)
	}
	return replyChan
}

func (n *RaftNode) applyAppendEntries(sendEntryLen int, message *message.AppendEntriesReply, appendSuccesCount *int) {
	if message.Term > n.currentTerm() {
		n.setState(FollowerState)
		n.setTerm(message.Term)
		return
	}

	// peerId := int(message.PeerId)
	if !message.Success {
		// logIndex := int64(len(node.logs) - 1)
		// confilctIndex := Min(response.ConflictIndex, logIndex)
		// node.nextIndex[peerId] = Max(0, confilctIndex)
		return
	}

	// n.nextIndex[peerId] += int64(sendEntryLen)
	// n.matchIndex[peerId] = n.nextIndex[peerId] - 1

	*appendSuccesCount++
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
		err = fmt.Errorf("[RaftNode.processClusterEvent] invalid event type. type : %s", e.Type.String())
	}
	return result, err
}

func (n *RaftNode) onRequestVote(e event.Event) (*message.RequestVoteReply, error) {
	requestVoteMessage, ok := e.Message.(*message.RequestVote)
	if !ok {
		return nil, fmt.Errorf("[RaftNode.onRequestVote] can not convert to *message.RequestVoteReply. %v", e)
	}

	log.Printf("[RaftNode.onRequestVote] my term : %d, request : %s", n.currentState(), requestVoteMessage)

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
	log.Printf("[RaftNode.onAppendEntries]")
	appendEntriesMessage, ok := e.Message.(*message.AppendEntries)
	if !ok {
		return nil, fmt.Errorf("[RaftNode.onAppendEntries] can not convert to *message.RequestVoteReply. %v", e)
	}

	reply := &message.AppendEntriesReply{}

	// peer term is less than me. return false to peer
	if appendEntriesMessage.Term < n.currentTerm() {
		reply.Term = n.currentTerm()
		reply.Success = false
		return reply, nil
	}

	n.setState(FollowerState)
	n.setTerm(reply.Term)
	return reply, nil
}
