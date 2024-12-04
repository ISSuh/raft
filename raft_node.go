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

package raft

import (
	"context"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/service"
)

type Node struct {
	raftService *service.RaftService
}

func NewRaftNode(path string) (*Node, error) {
	c, err := config.NewRaftConfig(path)
	if err != nil {
		return nil, err
	}

	return &Node{
		raftService: service.NewRaftService(c.Raft),
	}, nil
}

func (n *Node) Serve(context context.Context) error {
	return n.raftService.Serve(context)
}

func (n *Node) Apply(command []byte) error {
	return n.raftService.Apply(command)
}

func (n *Node) AddPeerNode(address string) error {
	return nil
}

func (n *Node) RemovePeerNode(address string) error {
	return nil
}
