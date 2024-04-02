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

package cluster

import (
	"fmt"
	"log"

	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

type node struct {
	requester net.NodeRequester
	metadata  *message.NodeMetadata
}

type nodeMap map[int]*message.NodeMetadata

type nodeManager struct {
	nodes nodeMap
}

func NewNodeManager() nodeManager {
	return nodeManager{
		nodes: make(nodeMap),
	}
}

func (n *nodeManager) addNode(meta *message.NodeMetadata) error {
	id := int(meta.Id)
	n.nodes[id] = meta
	return nil
}

func (n *nodeManager) removeNode(nodeId int) error {
	_, exist := n.nodes[nodeId]
	if !exist {
		return fmt.Errorf("[%d] node not exist.", nodeId)
	}

	delete(n.nodes, nodeId)
	return nil
}

func (n *nodeManager) findNode(nodeId int) (*message.NodeMetadata, error) {
	node, exist := n.nodes[nodeId]
	if !exist {
		return nil, fmt.Errorf("[%d] node not exist.", nodeId)
	}
	return node, nil
}

func (n *nodeManager) nodeList() *message.NodeMetadataesList {
	list := &message.NodeMetadataesList{
		Nodes: make([]*message.NodeMetadata, 0),
	}

	for _, node := range n.nodes {
		log.Printf("[TEST]node : %+v\n", node)
		list.Nodes = append(list.Nodes, node)
	}
	return list
}
