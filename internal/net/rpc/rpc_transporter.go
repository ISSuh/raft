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
	gonet "net"
	"net/rpc"
	"strconv"
	"sync"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/logger"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

const (
	RpcNetworkProtocol = "tcp"
	RpcServerName      = "Raft"
)

type RpcTransporter struct {
	address    config.Address
	listener   gonet.Listener
	rpcServer  *rpc.Server
	rpcHandler RpcHandler

	quit chan bool
	wg   sync.WaitGroup
}

func NewRpcTransporter(address config.Address, handler RpcHandler) *RpcTransporter {
	return &RpcTransporter{
		address:    address,
		rpcServer:  rpc.NewServer(),
		rpcHandler: handler,
		quit:       make(chan bool),
	}
}

func (t *RpcTransporter) Serve(context context.Context) error {
	logger.Info("[RpcTransporter.Serve]\n")
	return t.serveRpcServer(context)
}

func (t *RpcTransporter) StopAndWait() {
	t.quit <- true
	t.wg.Wait()
}

func (t *RpcTransporter) serveRpcServer(context context.Context) error {
	err := t.rpcServer.RegisterName(RpcServerName, t.rpcHandler)
	if err != nil {
		return err
	}

	address := t.address.String()
	t.listener, err = gonet.Listen(RpcNetworkProtocol, address)
	if err != nil {
		return err
	}

	t.runServer(context)
	return nil
}

func (t *RpcTransporter) runServer(context context.Context) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		for {
			conn, err := t.listener.Accept()
			if err != nil {
				select {
				case <-context.Done():
					logger.Info("contex cancel\n")
					return
				case <-t.quit:
					logger.Info("quit\n")
					return
				default:
					continue
				}
			}

			t.wg.Add(1)
			go func() {
				t.rpcServer.ServeConn(conn)
				t.wg.Done()
			}()
		}
	}()
}

func (t *RpcTransporter) ConnectNode(node *message.NodeMetadata) (net.NodeRequester, error) {
	ip := node.Address.Ip
	port := strconv.Itoa(int(node.Address.Port))
	if len(port) <= 1 {
		return nil, fmt.Errorf("[RpcTransporter.ConnectPeerNode] invalid peer port. %d", node.Address.Port)
	}

	address := ip + ":" + port
	addr, err := gonet.ResolveTCPAddr(RpcNetworkProtocol, address)
	if err != nil {
		return nil, err
	}

	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}
	return NewNodeRequester(client), nil
}

func (t *RpcTransporter) ConnectCluster(address config.Address) (net.ClusterRequester, error) {
	addr, err := gonet.ResolveTCPAddr(RpcNetworkProtocol, address.String())
	if err != nil {
		return nil, err
	}

	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}
	return NewClusterRequester(client), nil
}
