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

package rpc

import (
	"context"
	"fmt"
	gorpc "net/rpc"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/net"
)

type NodeRpcTransporter struct {
	config      config.Config
	transporter net.Transporter
	rpcHandler  *NodeRpcHandler

	eventChannel chan event.Event
}

func NewNodeRpcTransporter(config config.Config, eventChannel chan event.Event) (*NodeRpcTransporter, error) {
	rpcHandler := NewNodeRpcHandler(eventChannel)

	rpcServer := gorpc.NewServer()
	err := rpcServer.RegisterName(RpcServerName, rpcHandler)
	if err != nil {
		return nil, err
	}

	return &NodeRpcTransporter{
		config:       config,
		rpcHandler:   rpcHandler,
		transporter:  NewRpcTransporter(config, rpcServer),
		eventChannel: eventChannel,
	}, nil
}

func (t *NodeRpcTransporter) Serve(context context.Context) error {
	fmt.Printf("[NodeRpcTransporter.Serve]\n")

	err := t.connectCluster()
	if err != nil {
		return err
	}

	return t.transporter.Serve(context)
}

func (t *NodeRpcTransporter) StopAndWait() {
	t.transporter.StopAndWait()
}

func (t *NodeRpcTransporter) connectCluster() error {
	fmt.Printf("[NodeRpcTransporter.connectCluster]\n")
	return nil
}
