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

type LogEntry struct {
	Term    uint64
	Command []byte
}

type ApplyEntry struct {
	Command []byte
}

type ApplyEntryReply struct {
	Success bool
}

type PeerNodeInfo struct {
	Id      int    `json:"id"`
	Address string `json:"address"`
}

type RegistPeerNodeReply struct {
	Regist bool
}

type RequestVoteArgs struct {
	Term        uint64
	CandidateId int
	// LastLogIndex int
	// LastLogTerm  int
}

type RequestVoteReply struct {
	Term        uint64
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term     uint64
	LeaderId int

	PrevLogIndex      int64
	PrevLogTerm       uint64
	Entries           []LogEntry
	LeaderCommitIndex int64
}

type AppendEntriesReply struct {
	Term    uint64
	Success bool

	PeerId int

	ConflictIndex int64
	ConflictTerm  uint64
}