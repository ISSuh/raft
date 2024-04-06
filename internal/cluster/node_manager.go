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
	"errors"
	"fmt"
	"time"

	"github.com/ISSuh/raft/internal/logger"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

const (
	HealthCheckTimer        = 500 * time.Millisecond
	HealthCheckMaxFailCount = 3
)

type HealthCheckFailCallback func(*message.NodeMetadata)

type node struct {
	Metadata              *message.NodeMetadata
	Requester             net.NodeRequester
	HealthCheckRetryCount int
}

type nodeMap map[int]*node

type nodeManager struct {
	nodes           nodeMap
	quitHealthCheck map[int]chan struct{}
	callback        HealthCheckFailCallback
}

func NewNodeManager(callback HealthCheckFailCallback) nodeManager {
	return nodeManager{
		nodes:           make(nodeMap),
		quitHealthCheck: make(map[int]chan struct{}),
		callback:        callback,
	}
}

func (n *nodeManager) addNode(meta *message.NodeMetadata, requester net.NodeRequester) error {
	id := int(meta.Id)
	n.nodes[id] = &node{
		Metadata:  meta,
		Requester: requester,
	}

	n.quitHealthCheck[id] = make(chan struct{})

	go n.backgroundHealthCheck(id)
	return nil
}

func (n *nodeManager) removeNode(id int) error {
	_, exist := n.nodes[id]
	if !exist {
		return fmt.Errorf("[%d] node not exist.", id)
	}

	n.quitHealthCheck[id] <- struct{}{}

	close(n.quitHealthCheck[id])
	delete(n.quitHealthCheck, id)
	delete(n.nodes, id)
	return nil
}

func (n *nodeManager) findNode(nodeId int) (*node, error) {
	node, exist := n.nodes[nodeId]
	if !exist {
		return nil, fmt.Errorf("[%d] node not exist.", nodeId)
	}
	return node, nil
}

func (n *nodeManager) nodeList() []*node {
	nodeList := []*node{}
	for _, node := range n.nodes {
		nodeList = append(nodeList, node)
	}
	return nodeList
}

func (n *nodeManager) backgroundHealthCheck(id int) {
	ticker := time.NewTicker(HealthCheckTimer)
	for {
		select {
		case <-n.quitHealthCheck[id]:
			return
		case <-ticker.C:
			if err := n.nodeHealthChecking(id); err != nil {
				return
			}
		}
	}
}

func (n *nodeManager) nodeHealthChecking(id int) error {
	node, exist := n.nodes[id]
	if !exist {
		return fmt.Errorf("[nodeManager.nodeHealthChecking] node not exist. id : %d", id)
	}

	if err := node.Requester.HelthCheck(); err != nil {
		if node.HealthCheckRetryCount < HealthCheckMaxFailCount {
			node.HealthCheckRetryCount++
			logger.Info(
				"[nodeManager.nodeHealthChecking] %d node healcheck fall. retry [%d/%d] %s",
				id, node.HealthCheckRetryCount, HealthCheckMaxFailCount, err.Error(),
			)
			return nil
		} else {
			node.Requester.Close()
			close(n.quitHealthCheck[id])
			delete(n.quitHealthCheck, id)
			delete(n.nodes, id)

			n.callback(node.Metadata)
			return errors.Join(
				err,
				fmt.Errorf("[nodeManager.nodeHealthChecking] %d node healcheck fall and retry over %d. will disconnect.",
					id, HealthCheckMaxFailCount,
				),
			)
		}
	}

	node.HealthCheckRetryCount = 0
	return nil
}
