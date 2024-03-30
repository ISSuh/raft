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

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/net"
	"github.com/ISSuh/raft/internal/net/rpc"
)

type RaftCluster struct {
	config      config.Config
	manager     nodeManager
	transporter net.Transporter

	eventChannel chan event.Event
}

func NewRaftCluster(config config.Config) (*RaftCluster, error) {
	eventChannel := make(chan event.Event)
	transporter, err := rpc.NewClusterRpcTransporter(config, eventChannel)
	if err != nil {
		return nil, err
	}

	return &RaftCluster{
		config:       config,
		manager:      NewNodeManager(),
		transporter:  transporter,
		eventChannel: eventChannel,
	}, nil
}

func (c *RaftCluster) Serve(context context.Context) error {
	return c.transporter.Serve(context)
}

func (c *RaftCluster) Stop() {
	c.transporter.StopAndWait()
}

func (c *RaftCluster) eventLoop() {
}