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
	"net"
	"net/rpc"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	RpcServerName = "Raft"
)

type RpcProxy struct {
	node           *RaftNode
	transporter 	 *RpcTransporter
}

func (proxy *RpcProxy) RegistPeerNode(args PeerNodeInfo, reply *RegistPeerNodeReply) error {
	log.WithField("network", "network.RegistPeerNode").Info(goidForlog())
	err := proxy.transporter.ConnectToPeer(
		PeerNodeInfo{
			Id:      args.Id,
			Address: args.Address,
		})

	reply.Regist = (err == nil)
	return err
}

func (proxy *RpcProxy) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	log.WithField("network", "network.RequestVote").Info(goidForlog())
	proxy.node.onRequestVote(args, reply)
	return nil
}

func (proxy *RpcProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	// log.WithField("network", "network.AppendEntries").Info(goidForlog())
	proxy.node.onAppendEntries(args, reply)
	return nil
}

func (proxy *RpcProxy) ApplyEntry(args ApplyEntry, reply *ApplyEntryReply) error {
	log.WithField("network", "network.ApplyEntry").Info(goidForlog())
	reply.Success = proxy.node.ApplyEntry(args.Command)
	return nil
}

type RpcTransporter struct {
	running bool
	node *RaftNode

	listener  net.Listener
	address string

	rpcServer *rpc.Server
	rpcProxy  *RpcProxy
	quitSinal chan interface{}

	mutex sync.Mutex
	wg    sync.WaitGroup
}

func NewRpcTransport(node *RaftNode) *RpcTransporter {
	rpcProxy := &RpcProxy{node: node, transporter: nil}
	transporter := &RpcTransporter{
		running: false,
		node: node,
		rpcServer: rpc.NewServer(),
		rpcProxy: rpcProxy,
	}

	rpcProxy.transporter = transporter
	return transporter
}

func (rpcTransporter *RpcTransporter) RegistHandler(handler *NetworkHandler) {
}

func (rpcTransporter *RpcTransporter) Serve(address string) error {
	err := rpcTransporter.rpcServer.RegisterName(RpcServerName, rpcTransporter.rpcProxy)
	if err != nil {
		return err
	}

	rpcTransporter.listener, err = net.Listen("tcp", address)
	if err != nil {
		log.WithField("network", "network.Serve").Fatal(err)
		return err
	}

	log.WithField("network", "network.Serve").Info(goidForlog()+"listening at ", rpcTransporter.listener.Addr())

	rpcTransporter.wg.Add(1)
	go func() {
		defer rpcTransporter.wg.Done()

		for {
			conn, err := rpcTransporter.listener.Accept()
			if err != nil {
				select {
				case <-rpcTransporter.quitSinal:
					return
				default:
					log.WithField("network", "network.Serve").Fatal(goidForlog()+"accept error:", err)
					continue
				}
			}

			log.WithField("network", "network.Serve").Info(goidForlog()+"connected : ", conn.RemoteAddr().String())
			rpcTransporter.wg.Add(1)
			go func() {
				rpcTransporter.rpcServer.ServeConn(conn)
				rpcTransporter.wg.Done()
			}()
		}
	}()
	return nil
}

func (rpcTransporter *RpcTransporter) ConnectToPeer(peerInfo PeerNodeInfo) error {
	log.WithField("network", "network.ConnectToPeer").Info(goidForlog()+"peer : ", peerInfo)
	peerId := peerInfo.Id
	addr, err := net.ResolveTCPAddr("tcp", peerInfo.Address)
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	if peer := rpcTransporter.node.getPeer(peerId); peer != nil {
		log.WithField("network", "network.ConnectToPeer").Warn(goidForlog()+"alread registed. ", peerId)
		return nil
	}

	// connect rpc server
	rpcTransporter.mutex.Lock()
	defer rpcTransporter.mutex.Unlock()
	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	// regist5 peer
	peerNode := &RaftPeerNode{
		id:      peerInfo.Id,
		address: peerInfo.Address,
		client:  client,
	}

	rpcTransporter.node.addPeer(peerInfo.Id, peerNode)

	myInfo := PeerNodeInfo{
		Id:      rpcTransporter.node.id,
		Address: rpcTransporter.address,
	}

	// notify peer for regist me
	var reply RegistPeerNodeReply
	err = peerNode.RegistPeerNode(myInfo, &reply)
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	log.WithField("network", "network.ConnectToPeer").Info(goidForlog()+"result : ", reply.Regist)
	return nil
}

func (rpcTransporter *RpcTransporter) IsRunning() bool {
	return rpcTransporter.running
}

func (rpcTransporter *RpcTransporter) Stop() {
	rpcTransporter.quitSinal <- true
	rpcTransporter.running = false
	rpcTransporter.wg.Wait()
}
