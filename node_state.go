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
	return &NodeState{state: FOLLOWER}
}

func (nodeState *NodeState) currentState() State {
	return State(atomic.LoadUint32((*uint32)(&nodeState.state)))
}

func (nodeState *NodeState) compare(state State) bool {
	return nodeState.currentState() == state
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
