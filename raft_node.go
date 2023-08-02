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

	logs        []LogEntry
	commitIndex int64
	nextIndex   map[int]int64
	matchIndex  map[int]int64

	requestVoteSignal      chan RequestVoteArgs
	requestVoteReplySignal chan RequestVoteReply

	appendEntriesSignal      chan AppendEntriesArgs
	appendEntriesReplySignal chan AppendEntriesReply

	peerMutex sync.Mutex
	logMutex  sync.Mutex
	workGroup sync.WaitGroup
}

func NewRafeNode(id int) *RaftNode {
	return &RaftNode{
		NodeState: NewNodeState(),
		id:        id,

		peers:           make(map[int]*RaftPeerNode),
		timeoutDuration: DefaultElectionMinTimeout,
		stopped:         make(chan bool),

		logs:        make([]LogEntry, 0),
		commitIndex: -1,
		nextIndex:   make(map[int]int64),
		matchIndex:  make(map[int]int64),

		requestVoteSignal:        make(chan RequestVoteArgs, 512),
		requestVoteReplySignal:   make(chan RequestVoteReply, 512),
		appendEntriesSignal:      make(chan AppendEntriesArgs, 512),
		appendEntriesReplySignal: make(chan AppendEntriesReply, 512),
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

	for node.currentState() == FOLLOWER {
		timeout := timer(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)

		select {
		case <-node.stopped:
			node.setState(STOP)
			return
		case <-timeout:
			log.WithField("node", "node.follwerWork").Info(goidForlog() + "to be candidate")
			node.setState(CANDIDATE)
			return
		case request := <-node.requestVoteSignal:
			log.WithField("node", "node.follwerWork").Info(goidForlog() + " on Requse vote")

			respone := RequestVoteReply{Term: 0, VoteGranted: false}
			if request.Term > node.currentTerm() {
				node.setTerm(request.Term)

				respone.Term = node.currentTerm()
				respone.VoteGranted = true
			} else {
				respone.Term = node.currentTerm()
				respone.VoteGranted = false
			}

			node.requestVoteReplySignal <- respone
		case arg := <-node.appendEntriesSignal:
			node.handleOnAppendEntries(arg)
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
					if node.currentState() != CANDIDATE {
						return
					}

					var reply RequestVoteReply
					err := peer.RequestVote(arg, &reply)
					if err != nil {
						log.WithField("node", "node.candidateWork").Error(goidForlog()+"err : ", err)
						node.removePeer(peer.id)
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

		select {
		case <-node.stopped:
			node.setState(STOP)
			return
		case <-timeout:
			log.WithField("node", "node.candidateWork").Info(goidForlog() + "timeout. retry request vote")
			doErction = true
			electionGrantedCount = 0
			node.setTerm(node.currentTerm() - 1)
		case res := <-responses:
			if res.Term > node.currentTerm() {
				node.setState(FOLLOWER)
				return
			}

			if res.VoteGranted && res.Term == node.currentTerm() {
				electionGrantedCount++
			}
		}
	}
}

func (node *RaftNode) leaderWork() {
	log.WithField("node", "node.leaderWork").Info(goidForlog()+"current term : ", node.currentTerm())
	var timeout <-chan time.Time
	var responses chan *AppendEntriesReply
	doHeartBeat := true
	appendSuccesCount := 0

	for node.currentState() == LEADER {
		if doHeartBeat {
			log.WithField("node", "node.leaderWork").Info(goidForlog() + "heatbeat")
			responses = make(chan *AppendEntriesReply, len(node.peers))

			arg := AppendEntriesArgs{
				Term:              node.currentTerm(),
				LeaderId:          node.id,
				PrevLogIndex:      -1,
				PrevLogTerm:       0,
				Entries:           make([]LogEntry, 0),
				LeaderCommitIndex: node.commitIndex,
			}

			for _, peer := range node.peers {
				node.workGroup.Add(1)
				go func(peer *RaftPeerNode, arg AppendEntriesArgs) {
					defer node.workGroup.Done()

					nextIndex := node.nextIndex[peer.id]
					arg.PrevLogIndex = nextIndex - 1
					if arg.PrevLogIndex >= 0 {
						arg.PrevLogTerm = node.logs[arg.PrevLogIndex].Term
					}

					arg.Entries = append(arg.Entries, node.logs[nextIndex:]...)

					var reply AppendEntriesReply
					err := peer.AppendEntries(arg, &reply)
					if err != nil {
						log.WithField("node", "node.leaderWork").Error(goidForlog()+"err : ", err)
						node.removePeer(peer.id)
						return
					}

					responses <- &reply
				}(peer, arg)
			}

			doHeartBeat = false
			timeout = timer(DefaultHeartBeatMinTimeout, DefaultHeartBeatMaxTimeout)
		}

		majorityCount := (len(node.peers) / 2) + 1
		if appendSuccesCount == majorityCount {
			node.commitIndex = int64(len(node.logs) - 1)
			log.WithField("node", "node.leaderWork").Info(goidForlog()+" commit!!. index : ", node.commitIndex)
			return
		}

		select {
		case <-node.stopped:
			node.setState(STOP)
			return
		case <-timeout:
			doHeartBeat = true
		case res := <-responses:
			if res.Term > node.currentTerm() {
				node.setState(FOLLOWER)
				return
			}

			if res.Success && res.Term == node.currentTerm() {
				appendSuccesCount++
			}
		}
	}
}

func (node *RaftNode) addPeer(id int, peerNode *RaftPeerNode) {
	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()
	node.peers[id] = peerNode
}

func (node *RaftNode) removePeer(id int) {
	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()
	if peer := node.peers[id]; peer != nil {
		delete(node.peers, id)
	}
}

func (node *RaftNode) getPeer(id int) *RaftPeerNode {
	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()
	if peer := node.peers[id]; peer != nil {
		return peer
	}
	return nil
}

func (node *RaftNode) onRequestVote(args RequestVoteArgs, reply *RequestVoteReply) {
	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()
	node.requestVoteSignal <- args

	response := <-node.requestVoteReplySignal
	*reply = response
}

func (node *RaftNode) onAppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) {
	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()
	node.appendEntriesSignal <- args

	response := <-node.appendEntriesReplySignal
	*reply = response
}

func (node *RaftNode) handleOnAppendEntries(arg AppendEntriesArgs) {
	log.WithField("node", "node.follwerWork").Info(goidForlog()+"on appendEntry. ", arg)

	if arg.Term < node.currentTerm() {
		node.appendEntriesReplySignal <- AppendEntriesReply{Term: node.currentTerm(), Success: false}
		return
	}

	node.setTerm(arg.Term)
	respone := AppendEntriesReply{Term: node.currentTerm(), Success: true}
	node.appendEntriesReplySignal <- respone
}

func (node *RaftNode) ApplyEntry(command []byte) bool {
	node.logMutex.Lock()
	defer node.logMutex.Unlock()

	if node.currentState() != LEADER {
		return false
	}

	node.logs = append(node.logs, LogEntry{Term: node.currentTerm(), Command: command})
	return true
}
