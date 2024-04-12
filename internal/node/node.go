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
	"fmt"
	"sync/atomic"

	"github.com/ISSuh/raft/internal/message"
)

type State uint32

const (
	StopState State = iota
	FollowerState
	CandidateState
	LeaderState
)

func (s State) String() string {
	switch s {
	case StopState:
		return "Stop"
	case FollowerState:
		return "Follower"
	case CandidateState:
		return "Candidate"
	case LeaderState:
		return "Leader"
	}
	return "invalid state"
}

type Node struct {
	meta     *message.NodeMetadata
	state    State
	term     uint64
	leaderId int32
}

func NewNode(metadata *message.NodeMetadata) *Node {
	return &Node{
		meta:     metadata,
		state:    FollowerState,
		term:     0,
		leaderId: -1,
	}
}

func (n *Node) String() string {
	return fmt.Sprintf("Node{meta:%s, state:%s, term:%d}", n.meta, n.state, n.term)
}

func (n *Node) currentState() State {
	return State(atomic.LoadUint32((*uint32)(&n.state)))
}

func (n *Node) setState(state State) {
	atomic.StoreUint32((*uint32)(&n.state), uint32(state))
}

func (n *Node) currentTerm() uint64 {
	return atomic.LoadUint64(&n.term)
}

func (n *Node) setTerm(term uint64) {
	atomic.StoreUint64(&n.term, term)
}

func (n *Node) increaseTerm() {
	atomic.AddUint64(&n.term, 1)
}
