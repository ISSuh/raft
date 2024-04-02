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
	eventChannel := make(chan event.Event)
	handler := rpc.NewClusterRpcHandler(eventChannel)

	return &RaftCluster{
		config:       config,
		manager:      NewNodeManager(),
		transporter:  rpc.NewRpcTransporter(config.Cluster.Address, handler),
		eventChannel: eventChannel,
	}, nil
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
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return nil, fmt.Errorf("[RaftCluster.onNotifyMeToCluster] can not convert to *message.NodeMetadata. %v", e)
	}

	nodeList := c.manager.nodeList()
	c.manager.addNode(node)
	return nodeList, nil
}

func (c *RaftCluster) onDisconnect(e event.Event) error {
	log.Printf("[RaftCluster.onDisconnect]")
	node, ok := e.Message.(*message.NodeMetadata)
	if !ok {
		return fmt.Errorf("[RaftCluster.onDisconnect] can not convert to *message.NodeMetadata. %v", e)
	}

	c.manager.removeNode(int(node.Id))
	return nil
}

func (c *RaftCluster) onNodeList() (*message.NodeMetadataesList, error) {
	log.Printf("[RaftCluster.onNodeList]")
	return c.manager.nodeList(), nil
}
