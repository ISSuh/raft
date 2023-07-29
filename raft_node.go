package raft

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultElectionMinTimeout = 150 * time.Millisecond
	DefaultElectionMaxTimeout = 300 * time.Millisecond

	DefaultHeartBeatMinTimeout = 50 * time.Millisecond
	DefaultHeartBeatMaxTimeout = 150 * time.Millisecond
)

type RaftNode struct {
	*NodeState

	id    int
	peers map[int]*RaftPeerNode

	timeoutDuration time.Duration
	stopped         chan bool

	peerMutex sync.Mutex
	workGroup sync.WaitGroup
}

func NewRafeNode(id int) *RaftNode {
	return &RaftNode{
		NodeState:       NewNodeState(),
		id:              id,
		peers:           make(map[int]*RaftPeerNode),
		timeoutDuration: DefaultElectionMinTimeout,
		stopped:         make(chan bool),
	}
}

func (node *RaftNode) Run() {
	node.workGroup.Add(1)
	go func() {
		defer node.workGroup.Done()
		node.loop()
	}()
}

func (node *RaftNode) Stop() {
	node.setState(STOP)

	node.stopped <- true
	node.workGroup.Wait()
}

func (node *RaftNode) loop() {
	log.WithField("node", "node.loop").Info(goidForlog() + "run loop")

	state := node.currentState()
	for state != STOP {
		switch state {
		case FOLLOWER:
			node.follwerWork()
		case CANDIDATE:
			node.candidateWork()
		case LEADER:
			node.leaderWork()
		}
		state = node.currentState()
	}

	log.WithField("node", "node.loop").Info(goidForlog() + "loop end")
}

func (node *RaftNode) follwerWork() {
	timeout := timer(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)

	for node.currentState() == FOLLOWER {
		select {
		case <-node.stopped:
			node.setState(STOP)
			return
		case <-timeout:
			log.WithField("node", "node.follwerWork").Info(goidForlog() + "to be candidate")
			node.setState(CANDIDATE)
			return
		}
	}
}

func (node *RaftNode) candidateWork() {
	var timeout <-chan time.Time
	var responses chan *RequestVoteReply
	electionGrantedCount := 0
	doErction := true

	for node.currentState() == CANDIDATE {
		if doErction {
			node.increaseTerm()
			log.WithField("node", "node.candidateWork").Info(goidForlog()+"do erection. tern :  ", node.currentTerm())

			responses = make(chan *RequestVoteReply, len(node.peers))
			arg := RequestVoteArgs{Term: node.currentTerm(), CandidateId: node.id}
			for _, peer := range node.peers {

				node.workGroup.Add(1)
				go func(peer *RaftPeerNode, arg RequestVoteArgs) {
					defer node.workGroup.Done()

					var reply RequestVoteReply
					err := peer.RequestVote(arg, &reply)
					if err != nil {
						log.WithField("node", "node.candidateWork").Error(goidForlog()+"err : ", err)
						return
					}

					responses <- &reply
				}(peer, arg)
			}

			doErction = false
			electionGrantedCount++

			timeout = timer(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)
		}

		majorityCount := (len(node.peers) / 2) + 1
		if electionGrantedCount == majorityCount {
			log.WithField("node", "node.candidateWork").Info(goidForlog() + "to be leader")
			node.setState(LEADER)
			return
		}

		log.WithField("node", "node.candidateWork").Error(goidForlog() + "select!!!!")

		select {
		case <-node.stopped:
			node.setState(STOP)
			return
		case <-timeout:
			log.WithField("node", "node.candidateWork").Info(goidForlog() + "timeout. retry request vote")
			doErction = true
			electionGrantedCount = 0
		case res := <-responses:
			if res.VoteGranted && res.Term == node.currentTerm() {
				electionGrantedCount++
			}
		}
	}
}

func (node *RaftNode) leaderWork() {
	timeout := timer(DefaultHeartBeatMinTimeout, DefaultHeartBeatMaxTimeout)
	// var responses chan *AppendEntriesReply

	for node.currentState() == LEADER {
		select {
		case <-node.stopped:
			node.setState(STOP)
		case <-timeout:
			log.WithField("node", "node.leaderWork").Info(goidForlog() + "heatbeat")
			// responses = make(chan *AppendEntriesReply, len(node.peers))
			arg := RequestVoteArgs{Term: node.currentTerm(), CandidateId: node.id}
			for _, peer := range node.peers {

				node.workGroup.Add(1)
				go func(peer *RaftPeerNode, arg RequestVoteArgs) {
					defer node.workGroup.Done()

					var reply AppendEntriesReply
					err := peer.AppendEntries(arg, &reply)
					if err != nil {
						log.WithField("node", "node.leaderWork").Error(goidForlog()+"err : ", err)
						return
					}

					// responses <- &reply
				}(peer, arg)
			}

			timeout = timer(DefaultHeartBeatMinTimeout, DefaultHeartBeatMaxTimeout)
		}
	}
}

func (node *RaftNode) addPeer(id int, peerNode *RaftPeerNode) {
	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()
	node.peers[id] = peerNode
}
