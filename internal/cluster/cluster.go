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
	"context"
	"fmt"
	"log"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"github.com/ISSuh/raft/internal/net/rpc"
)

type RaftCluster struct {
	config      config.RaftConfig
	manager     nodeManager
	transporter net.Transporter

	eventChannel chan event.Event
}

func NewRaftCluster(config config.RaftConfig) (*RaftCluster, error) {
	e := make(chan event.Event)
	h := rpc.NewClusterRpcHandler(e)
	c := &RaftCluster{
		config:       config,
		transporter:  rpc.NewRpcTransporter(config.Cluster.Address, h),
		eventChannel: e,
	}

	c.manager = NewNodeManager(c.onHealthCheckFail)
	return c, nil
}

func (c *RaftCluster) Serve(context context.Context) error {
	if err := c.transporter.Serve(context); err != nil {
		return err
	}

	go c.eventLoop(context)
	return nil
}

func (c *RaftCluster) Stop() {
	c.transporter.StopAndWait()
}

func (c *RaftCluster) eventLoop(context context.Context) {
	for {
		select {
		case <-context.Done():
			log.Printf("[RaftCluster.eventLoog] context canceled")
			return
		case e := <-c.eventChannel:
			result, err := c.processEvent(e)
			if err != nil {
				log.Printf("[RaftCluster.eventLoog] err  %s", err.Error())
			}

			e.EventResultChannel <- &event.EventResult{
				Err:    err,
				Result: result,
			}
		}
	}
}

func (c *RaftCluster) processEvent(e event.Event) (interface{}, error) {
	var err error
	var result interface{}
	switch e.Type {
	case event.NotifyMeToCluster:
		result, err = c.onNotifyMeToCluster(e)
	case event.DeleteNode:
		err = c.onDisconnect(e)
	case event.NodeList:
		result, err = c.onNodeList()
	}
	return result, err
}

func (c *RaftCluster) onNotifyMeToCluster(e event.Event) (*message.NodeMetadataesList, error) {
	log.Printf("[RaftCluster.onNotifyMeToCluster]")
	newNode, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return nil, fmt.Errorf("[RaftCluster.onNotifyMeToCluster] can not convert to *message.NodeMetadata. %v", e)
	}

	// connect each other
	requester, err := c.transporter.ConnectNode(newNode)
	if err != nil {
		return nil, err
	}

	// notify new node connect to other node
	// make NodeMetadataesList message
	nodeListMessage := &message.NodeMetadataesList{
		Nodes: make([]*message.NodeMetadata, 0),
	}

	nodeList := c.manager.nodeList()
	for _, node := range nodeList {
		success, err := node.Requester.NotifyNodeConnected(newNode)
		if err != nil {
			return nil, err
		}

		if !success {
			return nil,
				fmt.Errorf(
					"[RaftCluster.onNotifyMeToCluster] fail NotifyNodeConnected. newnode : %d, node : %d",
					newNode.Id, node.Metadata.Id,
				)
		}

		nodeListMessage.Nodes = append(nodeListMessage.Nodes, node.Metadata)
	}

	// add node to node manager
	err = c.manager.addNode(newNode, requester)
	if err != nil {
		return nil, err
	}
	return nodeListMessage, nil
}

func (c *RaftCluster) onDisconnect(e event.Event) error {
	log.Printf("[RaftCluster.onDisconnect]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return fmt.Errorf("[RaftCluster.onDisconnect] can not convert to *message.NodeMetadata. %v", e)
	}

	if err := c.manager.removeNode(int(node.Id)); err != nil {
		return err
	}

	if err := c.notifyNodeDisconnectToAll(node); err != nil {
		return err
	}
	return nil
}

func (c *RaftCluster) onNodeList() (*message.NodeMetadataesList, error) {
	log.Printf("[RaftCluster.onNodeList]")
	list := &message.NodeMetadataesList{
		Nodes: make([]*message.NodeMetadata, 0),
	}

	nodeList := c.manager.nodeList()
	for _, node := range nodeList {
		list.Nodes = append(list.Nodes, node.Metadata)
	}
	return list, nil
}

func (c *RaftCluster) notifyNodeDisconnectToAll(disconnectedNode *message.NodeMetadata) error {
	log.Printf("[RaftCluster.notifyNodeDisconnectToAll]")

	var err error = nil
	nodeList := c.manager.nodeList()
	for _, node := range nodeList {
		if subErr := node.Requester.NotifyNodeDisconnected(disconnectedNode); subErr != nil {
			log.Printf("[RaftCluster.notifyNodeDisconnectToAll] %s", subErr.Error())
			err = subErr
		}
	}
	return err
}

func (c *RaftCluster) onHealthCheckFail(node *message.NodeMetadata) {
	log.Printf("[RaftCluster.onHealthCheckFail] node : %v", node)
	c.notifyNodeDisconnectToAll(node)
}
