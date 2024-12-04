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
	"sync"
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/logger"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/util"
)

type CandidateStateWorker struct {
	node            *Node
	timer           *time.Timer
	peerNodeManager *PeerNodeManager
	eventProcessor  event.EventProcessor
	workGroup       sync.WaitGroup
	quit            chan struct{}
}

func NewCandidateStateWorker(
	node *Node, peerNodeManager *PeerNodeManager, eventProcessor event.EventProcessor, quit chan struct{},
) Worker {
	return &CandidateStateWorker{
		node:            node,
		peerNodeManager: peerNodeManager,
		eventProcessor:  eventProcessor,
		quit:            quit,
	}
}

func (w *CandidateStateWorker) Work(c context.Context) {
	logger.Debug("[Work]")
	var replyChan chan *message.RequestVoteReply
	electionGrantedCount := 0
	needEraction := true
	w.node.leaderId = -1

	for w.node.currentState() == CandidateState {
		if needEraction {
			w.node.increaseTerm()
			electionGrantedCount++
			w.node.leaderId = w.node.meta.Id

			replyChan = w.doEraction()

			timeout := util.Timout(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)
			logger.Info("[Work] timeout : %d", timeout)

			w.timer = time.NewTimer(timeout)
			needEraction = false
		}

		peerNodeLen := w.peerNodeManager.numberOfPeer()
		majorityCount := 0
		if peerNodeLen == 1 {
			majorityCount = 2
		} else {
			majorityCount = (peerNodeLen / 2) + 1
		}

		logger.Info(
			"[Work] peer node len : %d, electionGrantedCount : %d, majorityCount : %d",
			peerNodeLen, electionGrantedCount, majorityCount,
		)

		if electionGrantedCount == majorityCount {
			w.node.setState(LeaderState)
			return
		}

		select {
		case <-c.Done():
			logger.Info("[Work] context done")
			w.node.setState(StopState)
		case <-w.quit:
			logger.Info("[Work] force quit")
			w.node.setState(StopState)
		case <-w.timer.C:
			logger.Info("[Work] timeout")
			needEraction = true
			electionGrantedCount = 0
			w.node.setTerm(w.node.currentTerm() - 1)
		case reply := <-replyChan:
			if reply.Term > w.node.currentTerm() {
				w.node.setState(FollowerState)
				return
			}

			if reply.VoteGranted && reply.Term == w.node.currentTerm() {
				electionGrantedCount++
			}
		case e := <-w.eventProcessor.WaitUntilEmit():
			result, err := w.eventProcessor.ProcessEvent(e)
			if err != nil {
				logger.Info("[Work] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (w *CandidateStateWorker) doEraction() chan *message.RequestVoteReply {
	logger.Debug("[doEraction]")
	peersLen := w.peerNodeManager.numberOfPeer()
	replyCahn := make(chan *message.RequestVoteReply, peersLen)

	requestVoteMessage := &message.RequestVote{
		Term:        w.node.currentTerm(),
		CandidateId: w.node.meta.Id,
	}

	peerNodes := w.peerNodeManager.findAll()
	for _, peer := range peerNodes {
		w.workGroup.Add(1)
		go func(peer *RaftPeerNode, message *message.RequestVote) {
			defer w.workGroup.Done()
			if w.node.currentState() != CandidateState {
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
