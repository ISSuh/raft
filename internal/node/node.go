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
	"log"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
)

type RaftNode struct {
	*NodeState

	metadata        *message.NodeMetadata
	eventChan       chan event.Event
	peerNodeManager PeerNodeManager

	quit chan struct{}
}

func NewRaftNode(metadata *message.NodeMetadata, eventChan chan event.Event, peerNodeManager PeerNodeManager) *RaftNode {
	return &RaftNode{
		NodeState:       NewNodeState(),
		metadata:        metadata,
		eventChan:       eventChan,
		peerNodeManager: peerNodeManager,
	}
}

func (n *RaftNode) NodeMetaData() *message.NodeMetadata {
	return n.metadata
}

func (n *RaftNode) ConnectToPeerNode(peerNodes *message.NodeMetadataesList) error {
	for _, peerNode := range peerNodes.Nodes {
		peerNode, err := n.peerNodeManager.RegistPeerNode(peerNode)
		if err != nil {
			return err
		}

		success, err := peerNode.NotifyMeToPeerNode(n.metadata)
		if err != nil {
			return err
		}

		if !success {
			return fmt.Errorf("[RaftNode.ConnectToPeerNode] can not connect to peer node\n")
		}
	}
	return nil
}

func (n *RaftNode) Run(c context.Context) {
	go n.eventLoop(c)
}

func (n *RaftNode) Stop() {
	go func() {
		n.quit <- struct{}{}
	}()
}

func (n *RaftNode) eventLoop(c context.Context) {
	for {
		select {
		case <-c.Done():
			log.Printf("[RaftNode.eventLoop] context done\n")
		case <-n.quit:
			log.Printf("[RaftNode.eventLoop] force quit\n")
		case e := <-n.eventChan:
			result, err := n.processEvent(e)
			if err != nil {
				log.Printf("[RaftNode.eventLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (n *RaftNode) processEvent(e event.Event) (interface{}, error) {
	var result interface{}
	var err error
	switch e.Type {
	case event.NotifyMeToNode:
		result, err = n.onnNtifyMeToNode(e)
	default:
		result = nil
		err = fmt.Errorf("[RaftNode.processEvent] invalid event type. type : %s", e.Type.String())
	}
	return result, err
}

func (n *RaftNode) onnNtifyMeToNode(e event.Event) (interface{}, error) {
	log.Printf("[RaftNode.onnNtifyMeToNode]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return nil, fmt.Errorf("[RaftNode.onnNtifyMeToNode] can not convert to *message.NodeMetadata. %v", e)
	}

	peerNode, err := n.peerNodeManager.RegistPeerNode(node)
	if err != nil {
		return nil, err
	}

	success, err := peerNode.NotifyMeToPeerNode(n.metadata)
	if err != nil {
		return nil, err
	}
	return success, err
}
