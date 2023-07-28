package raft

type RaftPeerNode struct {
}

type RaftNode struct {
	NodeState
}

func NewRafeNode() *RaftNode {
	return &RaftNode{
		NodeState: NewNodeState(),
	}
}

func (node *RaftNode) Run() {
	go func() {

	}()
}

func (node *RaftNode) Stop() {
}

// func (cm *RaftNode) runElectionTimer() {
// 	timeoutDuration := cm.electionTimeout()
// 	cm.mu.Lock()
// 	termStarted := cm.currentTerm
// 	cm.mu.Unlock()
// 	cm.dlog("election timer started (%v), term=%d", timeoutDuration, termStarted)

// 	// This loops until either:
// 	// - we discover the election timer is no longer needed, or
// 	// - the election timer expires and this CM becomes a candidate
// 	// In a follower, this typically keeps running in the background for the
// 	// duration of the CM's lifetime.
// 	ticker := time.NewTicker(10 * time.Millisecond)
// 	defer ticker.Stop()
// 	for {
// 		<-ticker.C

// 		cm.mu.Lock()
// 		if cm.state != Candidate && cm.state != Follower {
// 			cm.dlog("in election timer state=%s, bailing out", cm.state)
// 			cm.mu.Unlock()
// 			return
// 		}

// 		if termStarted != cm.currentTerm {
// 			cm.dlog("in election timer term changed from %d to %d, bailing out", termStarted, cm.currentTerm)
// 			cm.mu.Unlock()
// 			return
// 		}

// 		// Start an election if we haven't heard from a leader or haven't voted for
// 		// someone for the duration of the timeout.
// 		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
// 			cm.startElection()
// 			cm.mu.Unlock()
// 			return
// 		}
// 		cm.mu.Unlock()
// 	}
// }
