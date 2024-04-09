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
	"time"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/message"
)

const (
	DefaultRequestTimneout = 5 * time.Second
)

type NodeRequester interface {
	CloseNodeRequester()

	// call by cluster
	HelthCheck() error
	NotifyNodeConnected(node *message.NodeMetadata) (bool, error)
	NotifyNodeDisconnected(node *message.NodeMetadata) error

	// call by node
	RequestVote(requestVoteMessage *message.RequestVote) (*message.RequestVoteReply, error)
	AppendEntries(appendEntriesMessaage *message.AppendEntries) (*message.AppendEntriesReply, error)
}

type ClusterRequester interface {
	CloseClusterRequester()

	// call by node
	NotifyMeToCluster(myNode *message.NodeMetadata) (*message.NodeMetadataesList, error)
	DisconnectNode(myNode *message.NodeMetadata) error
	NodeList() (*message.NodeMetadataesList, error)
}

type Transporter interface {
	Serve(context context.Context) error
	StopAndWait()
	ConnectNode(peerMode *message.NodeMetadata) (NodeRequester, error)
	ConnectCluster(address config.Address) (ClusterRequester, error)
}
