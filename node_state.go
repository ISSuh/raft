package raft

import "sync/atomic"

type State uint32

const (
	FOLLOWER State = iota
	CANDIDATE
	LEADER
)

type NodeState struct {
	state       State
	currentTerm uint64
}

func NewNodeState() NodeState {
	return NodeState{state: FOLLOWER}
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
