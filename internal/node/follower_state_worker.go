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
	"github.com/ISSuh/raft/internal/util"
)

type FollowerStateWorker struct {
	*Node
	nodeEventChan  chan event.Event
	eventProcessor event.EventProcessor
	timer          *time.Timer
	quit           chan struct{}
}

func NewFollowerStateWorker(
	node *Node, nodeEventChan chan event.Event, quit chan struct{}) Worker {
	return &FollowerStateWorker{
		Node: node,
		// node:          node,
		nodeEventChan: nodeEventChan,
		quit:          quit,
	}
}

func (w *FollowerStateWorker) Work(c context.Context) {
	logger.Debug("[followerStateLoop]")
	for w.currentState() == FollowerState {
		timeout := util.Timout(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)
		w.timer = time.NewTimer(timeout)

		select {
		case <-c.Done():
			logger.Info("[followerStateLoop] context done\n")
			w.setState(StopState)
		case <-w.quit:
			logger.Info("[followerStateLoop] force quit\n")
			w.setState(StopState)
		case <-w.timer.C:
			w.setState(CandidateState)
		case e := <-w.eventProcessor.WaitUntilEmit():
			result, err := w.eventProcessor.Process(e)
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
