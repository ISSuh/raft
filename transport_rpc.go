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
	"errors"
	"net"
	"net/rpc"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	RpcServerName = "Raft"
)

type RpcRequestor struct {
	client  *rpc.Client
}

func (resquestor *RpcRequestor) RegistPeerNode(arg *NodeInfo, reply *RegistPeerNodeReply) error {
	log.WithField("peer", "RaftPeerNode.RegistPeerNode").Info(goidForlog())
	method := "Raft.RegistPeerNode"
	return resquestor.client.Call(method, arg, reply)
}

func (resquestor *RpcRequestor) RequestVote(arg *RequestVoteArgs, reply *RequestVoteReply) error {
	log.WithField("peer", "RaftPeerNode.RequestVote").Info(goidForlog())
	method := "Raft.RequestVote"
	return resquestor.client.Call(method, arg, reply)
}

func (resquestor *RpcRequestor) AppendEntries(arg *AppendEntriesArgs, reply *AppendEntriesReply) error {
	// log.WithField("peer", "RaftPeerNode.AppendEntries").Info(goidForlog())
	method := "Raft.AppendEntries"
	return resquestor.client.Call(method, arg, reply)
}

type RpcTransporter struct {
	address string
	listener  net.Listener
	rpcServer *rpc.Server
	handler Responsor

	peers map[int]NodeInfo

	running bool
	quitSinal chan interface{}

	mutex sync.Mutex
	wg    sync.WaitGroup
}

func NewRpcTransporter(handler Responsor) *RpcTransporter {
	transporter := &RpcTransporter{
		rpcServer: rpc.NewServer(),
		handler: handler,
		peers: map[int]NodeInfo{},
		running: false,
	}
	return transporter
}

func (rpcTransporter *RpcTransporter) RegistHandler(handler *Responsor) {
	rpcTransporter.handler = *handler
}

func (rpcTransporter *RpcTransporter) Serve(address string) error {
	err := rpcTransporter.rpcServer.RegisterName(RpcServerName, rpcTransporter)
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

func (rpcTransporter *RpcTransporter) ConnectToPeer(peerInfo NodeInfo) (*RaftPeerNode, error) {
	log.WithField("network", "network.ConnectToPeer").Info(goidForlog()+"peer : ", peerInfo)
	peerId := peerInfo.Id
	addr, err := net.ResolveTCPAddr("tcp", peerInfo.Address)
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return nil, err
	}

	if _, exist := rpcTransporter.peers[peerId]; exist {
		log.WithField("network", "network.ConnectToPeer").Warn(goidForlog()+"alread registed. ", peerId)
		return nil, errors.New("already exsit peer node")
	}

	// connect rpc server
	rpcTransporter.mutex.Lock()
	defer rpcTransporter.mutex.Unlock()
	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return nil, err
	}

	// regist peer
	peerNode := &RaftPeerNode{
		id:      peerInfo.Id,
		address: peerInfo.Address,
		requestor:  &RpcRequestor{client},
	}

	rpcTransporter.peers[peerId] = peerInfo
	return peerNode, nil
}

func (rpcTransporter *RpcTransporter) Stop() {
	rpcTransporter.quitSinal <- true
	rpcTransporter.running = false
	rpcTransporter.wg.Wait()
}

func (rpcTransporter *RpcTransporter) RegistPeerNode(args *NodeInfo, reply *RegistPeerNodeReply) error {
	log.WithField("network", "network.RegistPeerNode").Info(goidForlog())

	peerNode, err := rpcTransporter.ConnectToPeer(*args)
	if peerNode != nil {
		rpcTransporter.handler.onRegistPeerNode(peerNode)
	}

	reply.Regist = (err == nil)
	return err
}

func (rpcTransporter *RpcTransporter) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) error {
	log.WithField("network", "network.RequestVote").Info(goidForlog())
	if rpcTransporter.handler == nil {
		return errors.New("invalid handler")
	}
	rpcTransporter.handler.onRequestVote(args, reply)
	return nil
}

func (rpcTransporter *RpcTransporter) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {
	log.WithField("network", "network.AppendEntries").Info(goidForlog())
	if rpcTransporter.handler == nil {
		return errors.New("invalid handler")
	}
	rpcTransporter.handler.onAppendEntries(args, reply)
	return nil
}
