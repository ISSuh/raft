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
	"sync"
	"sync/atomic"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/logger"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

type PeerNodeManager struct {
	nodeMetaData     *message.NodeMetadata
	nodes            sync.Map
	nodeNum          int32
	clusterEventChan chan event.Event
	transpoter       net.Transporter
	quit             chan struct{}
}

func NewPeerNodeManager(
	nodeMetaData *message.NodeMetadata, clusterEventChan chan event.Event, transpoter net.Transporter, quit chan struct{},
) *PeerNodeManager {
	return &PeerNodeManager{
		nodeMetaData:     nodeMetaData,
		nodes:            sync.Map{},
		nodeNum:          0,
		clusterEventChan: clusterEventChan,
		transpoter:       transpoter,
		quit:             quit,
	}
}

func (m *PeerNodeManager) registPeerNode(metadata *message.NodeMetadata) error {
	logger.Info("[registPeerNode] regist peer node. id : %d", metadata.Id)
	requester, err := m.transpoter.ConnectNode(metadata)
	if err != nil {
		return err
	}

	_, exist := m.nodes.Load(metadata.Id)
	if exist {
		return fmt.Errorf("[registPeerNode][node : %d] already exist.", metadata.Id)
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
			logger.Info("[removePeerNode] can not convert value to RaftPeerNode. value : %v", value)
		}

		nodes = append(nodes, node)
		return true
	})
	return nodes
}

func (m *PeerNodeManager) findPeerNode(id int32) (*RaftPeerNode, error) {
	value, exist := m.nodes.Load(id)
	if !exist {
		return nil, fmt.Errorf("[findPeerNode][node : %d] not found peer node.", id)
	}

	node, ok := value.(*RaftPeerNode)
	if !ok {
		return nil, fmt.Errorf("[findPeerNode] can not convert value to RaftPeerNode. value : %v", value)
	}
	return node, nil
}

func (m *PeerNodeManager) removePeerNode(id int32) {
	value, exist := m.nodes.Load(id)
	if !exist {
		logger.Info("[removePeerNode][node : %d] not found peer node.", id)
		return
	}

	node, ok := value.(*RaftPeerNode)
	if !ok {
		logger.Info("[removePeerNode] can not convert value to RaftPeerNode. value : %v", value)
	}

	node.Close()
	m.nodes.Delete(id)

	atomic.AddInt32(&m.nodeNum, -1)
}

func (m *PeerNodeManager) notifyDisconnectToAllPeerNode() {
	m.nodes.Range(func(key, value any) bool {
		node, ok := value.(*RaftPeerNode)
		if !ok {
			logger.Info("[notifyDisconnectToAllPeerNode] can not convert value to RaftPeerNode. value : %v", value)
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

func (m *PeerNodeManager) clusterEventLoop(c context.Context) {
	for {
		select {
		case <-c.Done():
			logger.Info("[clusterEventLoop] context done")
		case <-m.quit:
			logger.Info("[clusterEventLoop] force quit")
		case e := <-m.WaitUntilEmit():
			result, err := m.ProcessEvent(e)
			if err != nil {
				logger.Info("[clusterEventLoop] %s\n", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (m *PeerNodeManager) WaitUntilEmit() <-chan event.Event {
	return m.clusterEventChan
}

func (m *PeerNodeManager) ProcessEvent(e event.Event) (interface{}, error) {
	var result interface{}
	var err error
	switch e.Type {
	case event.NotifyNodeConnected:
		result, err = m.onNotifyNodeConnected(e)
	case event.NotifyNodeDisconnected:
		result, err = m.onNotifyNodeDisconnected(e)
	default:
		result = nil
		err = fmt.Errorf("[processClusterEvent] invalid event type. type : %s", e.Type.String())
	}
	return result, err
}

func (m *PeerNodeManager) onNotifyNodeConnected(e event.Event) (bool, error) {
	logger.Debug("[onNotifyNodeConnected]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return false, fmt.Errorf("[onNotifyNodeConnected] can not convert to *message.NodeMetadata. %v", e)
	}

	err := m.registPeerNode(node)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *PeerNodeManager) onNotifyNodeDisconnected(e event.Event) (bool, error) {
	logger.Debug("[onNotifyNodeDisconnected]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return false, fmt.Errorf("[onNotifyNodeDisconnected] can not convert to *message.NodeMetadata. %v", e)
	}

	logger.Info("[onNotifyNodeDisconnected] remove peer node.  %+v", node)
	m.removePeerNode(node.Id)
	return true, nil
}
