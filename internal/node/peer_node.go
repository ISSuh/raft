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
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

type RaftPeerNode struct {
	metadata  *message.NodeMetadata
	requestor net.NodeRequester
}

func NewRaftPeerNode(metadata *message.NodeMetadata, requestor net.NodeRequester) *RaftPeerNode {
	return &RaftPeerNode{
		metadata:  metadata,
		requestor: requestor,
	}
}

func (n *RaftPeerNode) Id() int32 {
	return n.metadata.Id
}

func (n *RaftPeerNode) NotifyNodeConnected(message *message.NodeMetadata) (bool, error) {
	return n.requestor.NotifyNodeConnected(message)
}

func (n *RaftPeerNode) NotifyNodeDisconnected(message *message.NodeMetadata) error {
	return n.requestor.NotifyNodeDisconnected(n.metadata)
}

func (n *RaftPeerNode) RequestVote(message *message.RequestVote) (*message.RequestVoteReply, error) {
	return n.requestor.RequestVote(message)
}

func (n *RaftPeerNode) AppendEntries(message *message.AppendEntries) (*message.AppendEntriesReply, error) {
	return n.requestor.AppendEntries(message)
}

func (n *RaftPeerNode) Close() {
	n.requestor.CloseNodeRequester()
}
