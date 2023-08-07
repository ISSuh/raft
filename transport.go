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

type Responsor interface {
	onRegistPeerNode(peer *RaftPeerNode)
	onRequestVote(args *RequestVoteArgs, reply *RequestVoteReply)
	onAppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply)
}

type Requestor interface {
	RegistPeerNode(arg *NodeInfo, reply *RegistPeerNodeReply) error
	RequestVote(arg *RequestVoteArgs, reply *RequestVoteReply) error
	AppendEntries(arg *AppendEntriesArgs, reply *AppendEntriesReply) error
}

type Transporter interface {
	RegistHandler(handler *Responsor)
	ConnectToPeer(peerInfo NodeInfo) (*RaftPeerNode, error)
	Serve(address string) error
	Stop()
}