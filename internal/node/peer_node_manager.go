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
	"fmt"

	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

type PeerNodeManager struct {
	nodeMetaData *message.NodeMetadata
	nodes        map[int32]*RaftPeerNode
	transpoter   net.Transporter
}

func NewPeerNodeManager(nodeMetaData *message.NodeMetadata, transpoter net.Transporter) PeerNodeManager {
	return PeerNodeManager{
		nodeMetaData: nodeMetaData,
		nodes:        map[int32]*RaftPeerNode{},
		transpoter:   transpoter,
	}
}

func (m *PeerNodeManager) RegistPeerNode(metadata *message.NodeMetadata) error {
	requester, err := m.transpoter.ConnectNode(metadata)
	if err != nil {
		return err
	}

	_, exist := m.nodes[metadata.Id]
	if exist {
		return fmt.Errorf("[PeerNodeManager.RegistPeerNode] node already exist. id : %d", metadata.Id)
	}

	node := NewRaftPeerNode(metadata, requester)
	m.nodes[metadata.Id] = node
	return nil
}

func (m *PeerNodeManager) FindPeerNode(id int) (*RaftPeerNode, error) {
	key := int32(id)
	node, ok := m.nodes[key]
	if !ok {
		return nil, fmt.Errorf("[PeerNodeManager.RemovePeerNode] ]not found peer node. id : %d", id)
	}
	return node, nil
}

func (m *PeerNodeManager) RemovePeerNode(id int) {
	key := int32(id)
	m.nodes[key].Close()
	delete(m.nodes, key)
}

func (m *PeerNodeManager) NotifyDisconnectToAllPeerNode() {
	for _, node := range m.nodes {
		node.NotifyNodeDisconnected(m.nodeMetaData)
		node.Close()
	}
	m.nodes = make(map[int32]*RaftPeerNode)
}
