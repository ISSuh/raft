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

type LeaderStateWorker struct {
	*Node
	raftNode        *RaftNode
	timer           *time.Timer
	peerNodeManager *PeerNodeManager
	eventProcessor  event.EventProcessor
	workGroup       sync.WaitGroup
	quit            chan struct{}
}

func NewLeaderStateWorker(
	node *Node, raftNode *RaftNode, peerNodeManager *PeerNodeManager, eventProcessor event.EventProcessor, quit chan struct{},
) Worker {
	return &LeaderStateWorker{
		Node:            node,
		raftNode:        raftNode,
		peerNodeManager: peerNodeManager,
		eventProcessor:  eventProcessor,
		quit:            quit,
	}
}

func (w *LeaderStateWorker) Work(c context.Context) {
	logger.Debug("[Work]")

	// var timeout <-chan time.Time
	var replyChan chan *message.AppendEntriesReply
	needHeartBeat := true
	appendSuccesCount := 0

	for w.currentState() == LeaderState {
		if needHeartBeat {
			peersLen := w.peerNodeManager.numberOfPeer()
			replyChan = make(chan *message.AppendEntriesReply, peersLen)
			w.doHeartBeat(replyChan)

			if w.timer != nil {
				w.timer.Stop()
			}

			timeout := util.Timout(DefaultHeartBeatMinTimeout, DefaultHeartBeatMaxTimeout)
			w.timer = time.NewTimer(timeout)
			needHeartBeat = false
		}

		select {
		case <-c.Done():
			logger.Info("[Work] context done\n")
			w.setState(StopState)
		case <-w.quit:
			logger.Info("[Work] force quit\n")
			w.setState(StopState)
		case <-w.timer.C:
			needHeartBeat = true
		case reply := <-replyChan:
			if !w.applyAppendEntries(reply, &appendSuccesCount) {
				continue
			}

			poerNodeNum := w.peerNodeManager.numberOfPeer()
			majorityCount := (poerNodeNum / 2) + 1
			if appendSuccesCount == majorityCount {
				logLen := len(w.raftNode.entries)
				w.raftNode.commitIndex = int64(logLen - 1)
			}
		case e := <-w.eventProcessor.WaitUntilEmit():
			result, err := w.eventProcessor.ProcessEvent(e)
			if err != nil {
				logger.Info("[Work] %s\n", err.Error())
			} else {
				if e.Type == event.AppendEntries {
					w.timer.Stop()
				}
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (w *LeaderStateWorker) doHeartBeat(replyChan chan *message.AppendEntriesReply) {
	logger.Debug("[doHeartBeat]")
	peerNodes := w.peerNodeManager.findAll()
	for _, peer := range peerNodes {
		w.workGroup.Add(1)
		go func(peer *RaftPeerNode, replyChan chan *message.AppendEntriesReply) {
			defer w.workGroup.Done()

			appendEntriesMessage := &message.AppendEntries{
				Term:              w.currentTerm(),
				LeaderId:          w.leaderId,
				PrevLogIndex:      -1,
				PrevLogTerm:       0,
				Entries:           make([]*message.LogEntry, 0),
				LeaderCommitIndex: w.raftNode.commitIndex,
			}

			nextIndex := w.raftNode.nextIndex[peer.Id()]
			prevIndex := nextIndex - 1
			appendEntriesMessage.PrevLogIndex = prevIndex
			if prevIndex >= 0 {
				appendEntriesMessage.PrevLogTerm = w.raftNode.entries[prevIndex].Term
			}

			w.raftNode.logMutex.Lock()
			appendEntriesMessage.Entries = w.raftNode.entries[nextIndex:]
			w.raftNode.logMutex.Unlock()

			reply, err := peer.AppendEntries(appendEntriesMessage)
			if err != nil {
				logger.Info("[doHeartBeat] %s", err.Error())
				return
			}

			replyChan <- reply
		}(peer, replyChan)
	}
}

func (w *LeaderStateWorker) applyAppendEntries(message *message.AppendEntriesReply, appendSuccesCount *int) bool {
	logger.Debug("[applyAppendEntries]")
	if message.Term > w.currentTerm() {
		w.setState(FollowerState)
		w.setTerm(message.Term)
		return false
	}

	peerId := message.PeerId
	if !message.Success {
		logIndex := int64(len(w.raftNode.entries) - 1)
		confilctIndex := util.Min(message.ConflictIndex, logIndex)
		w.raftNode.nextIndex[peerId] = util.Max(0, confilctIndex)
		return false
	}

	w.raftNode.nextIndex[peerId] += message.ApplyEntriesLen
	w.raftNode.matchIndex[peerId] = w.raftNode.nextIndex[peerId] - 1

	*appendSuccesCount++
	return true
}
