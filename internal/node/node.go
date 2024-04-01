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
	"context"
	"fmt"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
)

type RaftNode struct {
	metadata  *message.NodeMetadata
	eventChan chan event.Event
}

func NewRaftNode(metadata *message.NodeMetadata, eventChan chan event.Event) *RaftNode {
	return &RaftNode{
		metadata:  metadata,
		eventChan: eventChan,
	}
}

func (n *RaftNode) NodeMetaData() *message.NodeMetadata {
	return n.metadata
}

func (n *RaftNode) Run(c context.Context) {
	go n.eventLoop(c)
}

func (n *RaftNode) eventLoop(c context.Context) {
	for {
		select {
		case <-c.Done():
			fmt.Printf("[RaftNode.eventLoop] context done\n")
		case event := <-n.eventChan:
			fmt.Printf("[RaftNode.eventLoop] event : %s\n", event)
		}
	}
}
