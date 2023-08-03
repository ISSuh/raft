package raft

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultElectionMinTimeout = 150 * time.Millisecond
	DefaultElectionMaxTimeout = 300 * time.Millisecond

	DefaultHeartBeatMinTimeout = 100 * time.Millisecond
	DefaultHeartBeatMaxTimeout = 150 * time.Millisecond
)

type AppendEntriesResult struct {
	sendEntryLen int
	reply        *AppendEntriesReply
}
type RaftNode struct {
	*NodeState

	id       int
	leaderId int
	peers    map[int]*RaftPeerNode

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
		leaderId:  id,

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
		case arg := <-node.requestVoteSignal:
			node.handleOnRequestVote(arg)
		case arg := <-node.appendEntriesSignal:
			node.handleOnAppendEntries(arg)
		}
	}
}

func (node *RaftNode) candidateWork() {
	var timeout <-chan time.Time
	var responses *chan *RequestVoteReply
	electionGrantedCount := 0
	needEraction := true

	for node.currentState() == CANDIDATE {
		if needEraction {
			node.increaseTerm()
			electionGrantedCount++
			node.leaderId = node.id

			responses = node.doEraction()

			timeout = timer(DefaultElectionMinTimeout, DefaultElectionMaxTimeout)
			needEraction = false
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
		case <-timeout:
			log.WithField("node", "node.candidateWork").Info(goidForlog() + "timeout. retry request vote")
			needEraction = true
			electionGrantedCount = 0
			node.setTerm(node.currentTerm() - 1)
		case arg := <-node.requestVoteSignal:
			node.handleOnRequestVote(arg)
		case res := <-*responses:
			node.applyRequestVote(res, &electionGrantedCount)
		}
	}
}

func (node *RaftNode) leaderWork() {
	log.WithField("node", "node.leaderWork").Info(goidForlog()+"current term : ", node.currentTerm())

	var timeout <-chan time.Time
	var responses *chan *AppendEntriesResult
	needHeartBeat := true
	appendSuccesCount := 0

	for node.currentState() == LEADER {
		if needHeartBeat {
			responses = node.doHeartBeat()

			timeout = timer(DefaultHeartBeatMinTimeout, DefaultHeartBeatMaxTimeout)
			needHeartBeat = false
		}

		select {
		case <-node.stopped:
			node.setState(STOP)
		case <-timeout:
			needHeartBeat = true
		case result := <-*responses:
			node.applyAppendEntries(result.sendEntryLen, result.reply, &appendSuccesCount)

			majorityCount := (len(node.peers) / 2) + 1
			if appendSuccesCount == majorityCount {
				node.commitIndex = int64(len(node.logs) - 1)
			}
		}
	}
}

func (node *RaftNode) addPeer(id int, peerNode *RaftPeerNode) {
	log.WithField("node", "node.addPeer").Info(goidForlog()+" - id : ", id)

	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()

	node.peers[id] = peerNode
	node.nextIndex[id] = int64(len(node.logs))
	node.matchIndex[id] = -1
}

func (node *RaftNode) removePeer(id int) {
	log.WithField("node", "node.removePeer").Info(goidForlog()+" - id : ", id)

	node.peerMutex.Lock()
	defer node.peerMutex.Unlock()
	if peer := node.peers[id]; peer != nil {
		delete(node.peers, id)
		delete(node.nextIndex, id)
		delete(node.matchIndex, id)
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

func (node *RaftNode) doEraction() *chan *RequestVoteReply {
	log.WithField("node", "node.doEraction").Info(goidForlog()+"do erection. tern :  ", node.currentTerm())
	responses := make(chan *RequestVoteReply, len(node.peers))
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
				log.WithField("node", "node.doEraction").Error(goidForlog()+"err : ", err)
				node.removePeer(peer.id)
				return
			}

			responses <- &reply
		}(peer, arg)
	}
	return &responses
}

func (node *RaftNode) doHeartBeat() *chan *AppendEntriesResult {
	log.WithField("node", "node.doHeartBeat").Info(goidForlog() + "heatbeat")
	responses := make(chan *AppendEntriesResult, len(node.peers))

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

			node.logMutex.Lock()
			arg.Entries = node.logs[nextIndex:]
			node.logMutex.Unlock()

			var reply AppendEntriesReply
			err := peer.AppendEntries(arg, &reply)
			if err != nil {
				log.WithField("node", "node.leaderWork").Error(goidForlog()+"err : ", err)
				node.removePeer(peer.id)
				return
			}

			responses <- &AppendEntriesResult{len(arg.Entries), &reply}
		}(peer, arg)
	}
	return &responses
}

func (node *RaftNode) handleOnRequestVote(arg RequestVoteArgs) {
	log.WithField("node", "node.handleOnRequestVote").Info(goidForlog() + " on Requse vote")

	respone := RequestVoteReply{Term: 0, VoteGranted: false}
	if arg.Term > node.currentTerm() {
		node.setTerm(arg.Term)
		node.setState(FOLLOWER)
		node.leaderId = arg.CandidateId

		respone.VoteGranted = true
	} else {
		respone.VoteGranted = false
	}

	respone.Term = node.currentTerm()

	node.requestVoteReplySignal <- respone
}

func (node *RaftNode) handleOnAppendEntries(arg AppendEntriesArgs) {
	if len(arg.Entries) > 0 {
		log.WithField("node", "node.handleOnAppendEntries").Info(goidForlog()+"on appendEntry. ", arg)
	}

	response := AppendEntriesReply{
		Term:          node.currentTerm(),
		Success:       false,
		PeerId:        node.id,
		ConflictIndex: -1,
		ConflictTerm:  0,
	}

	if arg.Term < node.currentTerm() {
		node.appendEntriesReplySignal <- response
		return
	}

	node.setState(FOLLOWER)
	node.setTerm(arg.Term)

	if !node.isValidPrevLogIndexAndTerm(arg.PrevLogTerm, arg.PrevLogIndex) {
		var conflictIndex int64 = 0
		var conflictTerm uint64 = 0

		if conflictIndex >= int64(len(node.logs)) {
			conflictIndex = int64(len(node.logs))
			conflictTerm = 0
		} else {
			logIndex := int64(len(node.logs)) - 1
			conflictIndex = Min(arg.PrevLogIndex, logIndex)
			conflictTerm = node.logs[conflictIndex].Term

			for {
				if conflictIndex <= node.commitIndex {
					break
				}

				if node.logs[conflictIndex].Term != conflictTerm {
					break
				}

				conflictIndex--
			}

			conflictIndex = Max(node.commitIndex+1, conflictIndex)
			conflictTerm = node.logs[conflictIndex].Term
		}
		response.ConflictIndex = conflictIndex
		response.ConflictTerm = conflictTerm
		node.appendEntriesReplySignal <- response
		return
	}

	logInsertIndex := arg.PrevLogIndex + 1
	newEntriesIndex := 0
	for {
		if (logInsertIndex >= int64(len(node.logs))) || newEntriesIndex >= len(arg.Entries) {
			break
		}

		if node.logs[logInsertIndex].Term != arg.Entries[newEntriesIndex].Term {
			break
		}

		logInsertIndex++
		newEntriesIndex++
	}

	if newEntriesIndex < len(arg.Entries) {
		node.logs = append(node.logs[:logInsertIndex], arg.Entries[newEntriesIndex:]...)
	}

	if arg.LeaderCommitIndex > node.commitIndex {
		logIndex := int64(len(node.logs) - 1)
		node.commitIndex = Min(arg.LeaderCommitIndex, logIndex)
		// need commit
	}

	response.Success = true
	node.appendEntriesReplySignal <- response
}

func (node *RaftNode) applyRequestVote(response *RequestVoteReply, electionGrantedCount *int) {
	if response.Term > node.currentTerm() {
		node.setState(FOLLOWER)
		return
	}

	if response.VoteGranted && response.Term == node.currentTerm() {
		*electionGrantedCount++
	}
}

func (node *RaftNode) applyAppendEntries(sendEntryLen int, response *AppendEntriesReply, appendSuccesCount *int) {
	if response.Term > node.currentTerm() {
		node.setState(FOLLOWER)
		node.setTerm(response.Term)
		return
	}

	peerId := response.PeerId
	if !response.Success {
		logIndex := int64(len(node.logs) - 1)
		confilctIndex := Min(response.ConflictIndex, logIndex)
		node.nextIndex[peerId] = Max(0, confilctIndex)
		return
	}

	node.nextIndex[peerId] += int64(sendEntryLen)
	node.matchIndex[peerId] = node.nextIndex[peerId] - 1

	*appendSuccesCount++
}

func (node *RaftNode) isValidPrevLogIndexAndTerm(prevLogTerm uint64, prevlogIndex int64) bool {
	if (prevlogIndex == -1) ||
		((prevlogIndex < int64(len(node.logs))) && (prevLogTerm == node.logs[prevlogIndex].Term)) {
		return true
	}
	return false
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
