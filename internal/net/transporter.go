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

package net

import (
	"context"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/message"
)

type Responsor interface {
	onConnectToPeer(peer *message.NodeMetadata)
	onRequestVote(args *message.RequestVote, reply *message.RequestVoteReply)
	onAppendEntries(args *message.AppendEntries, reply *message.AppendEntriesReply)
	onApplyEntry(args *message.ApplyEntry)
}

type Requestor interface {
	ConnectToPeer(arg *message.NodeMetadata, reply *bool) error
	RequestVote(arg *message.RequestVote, reply *message.RequestVoteReply) error
	AppendEntries(arg *message.AppendEntries, reply *message.AppendEntriesReply) error
}

type NodeRequester interface {
	HelthCheck() error
	NotifyMeToPeerNode(myNode *message.NodeMetadata) (bool, error)
	RequestVote(message *message.RequestVote) (*message.RequestVoteReply, error)
	AppendEntries(message *message.AppendEntries) (*message.AppendEntriesReply, error)
}

type ClusterRequester interface {
	NotifyMeToCluster(myNode *message.NodeMetadata) (*message.NodeMetadataesList, error)
	Disconnect(myNode *message.NodeMetadata) error
	NodeList() (*message.NodeMetadataesList, error)
}

type Transporter interface {
	Serve(context context.Context) error
	StopAndWait()
	ConnectPeerNode(peerMode *message.NodeMetadata) (NodeRequester, error)
	ConnectCluster(address config.Address) (ClusterRequester, error)
}
