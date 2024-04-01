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
	config      config.Config
	node        *node.RaftNode
	requester   net.ClusterRequester
	transporter net.Transporter
}

func NewRaftService(config config.Config) *RaftService {
	c := make(chan event.Event)
	h := rpc.NewNodeRpcHandler(c)
	t := rpc.NewRpcTransporter(config.Raft.Server.Address, h)

	metadata := &message.NodeMetadata{
		Id: int32(config.Raft.Server.Id),
		Address: &message.Address{
			Ip:   config.Raft.Server.Address.Ip,
			Port: int32(config.Raft.Server.Address.Port),
		},
	}

	node := node.NewRaftNode(metadata, c)
	s := &RaftService{
		config:      config,
		node:        node,
		transporter: t,
	}
	return s
}

func (s *RaftService) Run(context context.Context) error {
	err := s.transporter.Serve(context)
	if err != nil {
		return err
	}

	s.requester, err = s.transporter.ConnectCluster(s.config.Raft.Cluster.Address)
	if err != nil {
		return err
	}

	_, err = s.requester.NotifyMeToCluster(s.node.NodeMetaData())
	if err != nil {
		return err
	}

	return nil
}

func (s *RaftService) ConnectToPeers(peers map[int]string) error {
}

func (s *RaftService) ApplyEntries(entris [][]byte) {
}
