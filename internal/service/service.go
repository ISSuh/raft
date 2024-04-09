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

package service

import (
	"context"
	"fmt"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"github.com/ISSuh/raft/internal/net/rpc"
	"github.com/ISSuh/raft/internal/node"
)

type EntryHandler interface {
	EntryUpdated(log []byte)
}

type TestEntryHandler struct {
}

func (t *TestEntryHandler) EntryUpdated(log []byte) {
	fmt.Printf("[TEST] log : %s\n", string(log))
}

type RaftService struct {
	config      config.RaftConfig
	node        *node.RaftNode
	requester   net.ClusterRequester
	transporter net.Transporter
}

func NewRaftService(config config.RaftConfig) *RaftService {
	metadata := &message.NodeMetadata{
		Id: int32(config.Node.Id),
		Address: &message.Address{
			Ip:   config.Node.Address.Ip,
			Port: int32(config.Node.Address.Port),
		},
	}

	q := make(chan struct{}, node.BackgroundLoopLen)
	nc := make(chan event.Event)
	cc := make(chan event.Event)

	h := rpc.NewNodeRpcHandler(nc, cc, config.Node.Event.Timeout)
	t := rpc.NewRpcTransporter(config.Node.Address, config.Node.Transport, h)
	m := node.NewPeerNodeManager(metadata, cc, t, q)

	node := node.NewRaftNode(metadata, nc, m, q)
	s := &RaftService{
		config:      config,
		node:        node,
		transporter: t,
	}
	return s
}

func (s *RaftService) Serve(context context.Context) error {
	err := s.transporter.Serve(context)
	if err != nil {
		return err
	}

	s.node.Run(context)

	s.requester, err = s.transporter.ConnectCluster(s.config.Cluster.Address)
	if err != nil {
		s.node.Stop()
		return err
	}

	peerNodes, err := s.requester.NotifyMeToCluster(s.node.NodeMetaData())
	if err != nil {
		s.node.Stop()
		return err
	}

	if err := s.node.ConnectToPeerNode(peerNodes); err != nil {
		s.node.Stop()
		return err
	}
	return nil
}

func (s *RaftService) AppendLog(command []byte) error {
	return s.node.Submit(command)
}
