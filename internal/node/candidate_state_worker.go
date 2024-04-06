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
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/logger"
	"github.com/ISSuh/raft/internal/message"
)

type CandidateStateWorker struct {
	node          *RaftNode
	timer         *time.Timer
	nodeEventChan chan event.Event
	peerManager   *PeerNodeManager
	quit          chan struct{}
}

func NewCandidateStateWorkerWorker(
	node *RaftNode, nodeEventChan chan event.Event, quit chan struct{}) Worker {
	return &CandidateStateWorker{
		node:          node,
		nodeEventChan: nodeEventChan,
		quit:          quit,
	}
}

func (w *CandidateStateWorker) Work(c context.Context) {
}

func (w *CandidateStateWorker) doEraction() chan *message.RequestVoteReply {
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
