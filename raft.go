/*
MIT License

Copyright (c) 2023 ISSuh

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

package raft

import (
	"context"
	"fmt"

	"github.com/ISSuh/raft/message"
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
	config      Config
	node        *RaftNode
	transporter Transporter
	context     context.Context
	testBlock   chan bool
}

func NewRaftService(config Config) *RaftService {
	address := config.Raft.Server.Address.Ip + ":" + config.Raft.Server.Address.Port
	nodeInfo := NodeInfo{
		Id:      config.Raft.Server.Id,
		Address: address,
	}

	t := NewRpcTransporter()
	n := NewRaftNode(nodeInfo, t)
	t.RegistHandler(n)

	s := &RaftService{
		config:      config,
		node:        n,
		transporter: t,
		testBlock:   make(chan bool),
	}
	return s
}

func (s *RaftService) Node() *RaftNode {
	return s.node
}

func (s *RaftService) RegistEntryHandler(handler EntryHandler) {
	s.node.registEntryHandler(handler)
}

func (s *RaftService) Run(c context.Context) {
	s.context = c

	err := s.transporter.Serve(s.context, s.node.info.Address)
	if err != nil {
		return
	}
	s.node.Run(s.context)
}

func (s *RaftService) Stop() {
	s.transporter.Stop()
}

func (s *RaftService) ConnectToPeers(peers map[int]string) error {
	myInfo := message.RegistPeer{
		Id:      int32(s.node.info.Id),
		Address: s.node.info.Address,
	}

	for peerId, peerAddress := range peers {
		peerNode, err := s.transporter.RegistPeerNode(
			&message.RegistPeer{
				Id:      int32(peerId),
				Address: peerAddress,
			})
		if err != nil {
			continue
		}

		s.node.addPeer(peerNode.id, peerNode)

		var reply bool
		err = peerNode.ConnectToPeer(&myInfo, &reply)
		if err != nil || !reply {
			return err
		}
	}
	return nil
}

func (s *RaftService) ApplyEntries(entris [][]byte) {
	for _, entry := range entris {
		s.node.ApplyEntry(entry)
	}
}
