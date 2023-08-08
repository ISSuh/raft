/*
MIT License

Copyright (c) 2023 ISSuh

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

package raft

import (
	"sync/atomic"
)

type State uint32

const (
	STOP State = iota
	FOLLOWER
	CANDIDATE
	LEADER
)

type NodeState struct {
	state State
	term  uint64
}

func NewNodeState() *NodeState {
	return &NodeState{state: FOLLOWER, term: 0}
}

func (nodeState *NodeState) currentState() State {
	return State(atomic.LoadUint32((*uint32)(&nodeState.state)))
}

func (nodeState *NodeState) setState(state State) {
	atomic.StoreUint32((*uint32)(&nodeState.state), uint32(state))
}

func (nodeState *NodeState) currentTerm() uint64 {
	return atomic.LoadUint64(&nodeState.term)
}

func (nodeState *NodeState) setTerm(term uint64) {
	atomic.StoreUint64(&nodeState.term, term)
}

func (nodeState *NodeState) increaseTerm() {
	atomic.AddUint64(&nodeState.term, 1)
}
