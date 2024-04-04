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
	"log"
	"sync"
	"sync/atomic"

	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

type PeerNodeManager struct {
	nodeMetaData *message.NodeMetadata
	nodes        sync.Map
	nodeNum      int32
	transpoter   net.Transporter
}

// map[int32]*RaftPeerNode{}
func NewPeerNodeManager(nodeMetaData *message.NodeMetadata, transpoter net.Transporter) *PeerNodeManager {
	return &PeerNodeManager{
		nodeMetaData: nodeMetaData,
		nodes:        sync.Map{},
		nodeNum:      0,
		transpoter:   transpoter,
	}
}

func (m *PeerNodeManager) registPeerNode(metadata *message.NodeMetadata) error {
	requester, err := m.transpoter.ConnectNode(metadata)
	if err != nil {
		return err
	}

	_, exist := m.nodes.Load(metadata.Id)
	if exist {
		return fmt.Errorf("[PeerNodeManager.registPeerNode] node already exist. id : %d", metadata.Id)
	}

	node := NewRaftPeerNode(metadata, requester)
	m.nodes.Store(metadata.Id, node)

	atomic.AddInt32(&m.nodeNum, 1)
	return nil
}

func (m *PeerNodeManager) findAll() []*RaftPeerNode {
	nodes := []*RaftPeerNode{}
	m.nodes.Range(func(_, value any) bool {
		node, ok := value.(*RaftPeerNode)
		if !ok {
			log.Printf("[PeerNodeManager.removePeerNode] can not convert value to RaftPeerNode. value : %v\n", value)
		}

		nodes = append(nodes, node)
		return true
	})
	return nodes
}

func (m *PeerNodeManager) findPeerNode(id int) (*RaftPeerNode, error) {
	key := int32(id)
	value, exist := m.nodes.Load(key)
	if !exist {
		return nil, fmt.Errorf("[PeerNodeManager.findPeerNode] not found peer node. id : %d", id)
	}

	node, ok := value.(*RaftPeerNode)
	if !ok {
		return nil, fmt.Errorf("[PeerNodeManager.findPeerNode] can not convert value to RaftPeerNode. value : %v", value)
	}
	return node, nil
}

func (m *PeerNodeManager) removePeerNode(id int) {
	key := int32(id)
	value, exist := m.nodes.Load(key)
	if !exist {
		log.Printf("[PeerNodeManager.removePeerNode] not found peer node. id : %d\n", id)
		return
	}

	node, ok := value.(*RaftPeerNode)
	if !ok {
		log.Printf("[PeerNodeManager.removePeerNode] can not convert value to RaftPeerNode. value : %v\n", value)
	}

	node.Close()
	m.nodes.Delete(key)

	atomic.AddInt32(&m.nodeNum, 1)
}

func (m *PeerNodeManager) notifyDisconnectToAllPeerNode() {
	m.nodes.Range(func(key, value any) bool {
		node, ok := value.(*RaftPeerNode)
		if !ok {
			log.Printf("[PeerNodeManager.notifyDisconnectToAllPeerNode] can not convert value to RaftPeerNode. value : %v\n", value)
			return false
		}

		node.NotifyNodeDisconnected(m.nodeMetaData)
		node.Close()
		return true
	})

	// clear peer nodes
	m.nodes = sync.Map{}
	atomic.StoreInt32(&m.nodeNum, 0)
}

func (m *PeerNodeManager) numberOfPeer() int {
	num := atomic.LoadInt32(&m.nodeNum)
	return int(num)
}
