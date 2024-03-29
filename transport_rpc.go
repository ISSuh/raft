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
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"sync"

	"github.com/ISSuh/raft/message"
	log "github.com/sirupsen/logrus"
)

const (
	RpcServerName = "Raft"
)

type RpcRequestor struct {
	client *rpc.Client
}

func (resquestor *RpcRequestor) ConnectToPeer(arg *message.RegistPeer, reply *bool) error {
	log.WithField("RpcTransporter", "RaftPeerNode.ConnectToPeer").Info(goidForlog())
	method := "Raft.ConnectToPeer"
	return resquestor.client.Call(method, arg, reply)
}

func (resquestor *RpcRequestor) RequestVote(arg *message.RequestVote, reply *message.RequestVoteReply) error {
	log.WithField("RpcTransporter", "RaftPeerNode.RequestVote").Info(goidForlog())
	method := "Raft.RequestVote"
	return resquestor.client.Call(method, arg, reply)
}

func (resquestor *RpcRequestor) AppendEntries(arg *message.AppendEntries, reply *message.AppendEntriesReply) error {
	log.WithField("RpcTransporter", "RaftPeerNode.AppendEntries").Info(goidForlog())
	method := "Raft.AppendEntries"
	return resquestor.client.Call(method, arg, reply)
}

type RpcTransporter struct {
	listener  net.Listener
	rpcServer *rpc.Server
	handler   Responsor

	peers map[int]*message.RegistPeer

	running   bool
	quitSinal chan interface{}

	mutex sync.Mutex
	wg    sync.WaitGroup
}

func NewRpcTransporter() *RpcTransporter {
	transporter := &RpcTransporter{
		rpcServer: rpc.NewServer(),
		peers:     map[int]*message.RegistPeer{},
		running:   false,
	}
	return transporter
}

func (t *RpcTransporter) RegistHandler(handler Responsor) {
	t.handler = handler
}

func (t *RpcTransporter) Serve(c context.Context, address string) error {
	err := t.rpcServer.RegisterName(RpcServerName, t)
	if err != nil {
		return err
	}

	t.listener, err = net.Listen("tcp", address)
	if err != nil {
		log.WithField("RpcTransporter", "transporter.Serve").Fatal(err)
		return err
	}

	log.WithField("RpcTransporter", "transporter.Serve").Info(goidForlog()+"listening at ", t.listener.Addr())

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		for {
			conn, err := t.listener.Accept()
			if err != nil {
				select {
				case <-t.quitSinal:
					return
				case <-c.Done():
					fmt.Println("contex cancel")
					return
				default:
					log.WithField("RpcTransporter", "transporter.Serve").Fatal(goidForlog()+"accept error:", err)
					continue
				}
			}

			log.WithField("RpcTransporter", "transporter.Serve").Info(goidForlog()+"connected : ", conn.RemoteAddr().String())
			t.wg.Add(1)
			go func() {
				t.rpcServer.ServeConn(conn)
				t.wg.Done()
			}()
		}
	}()
	return nil
}

func (t *RpcTransporter) RegistPeerNode(peerInfo *message.RegistPeer) (*RaftPeerNode, error) {
	log.WithField("RpcTransporter", "transporter.RegistPeerNode").Info(goidForlog()+"peer : ", peerInfo.String())
	peerId := int(peerInfo.GetId())
	addr, err := net.ResolveTCPAddr("tcp", peerInfo.Address)
	if err != nil {
		log.WithField("RpcTransporter", "transporter.RegistPeerNode").Error(goidForlog()+"err : ", err)
		return nil, err
	}

	if _, exist := t.peers[peerId]; exist {
		log.WithField("RpcTransporter", "transporter.RegistPeerNode").Warn(goidForlog()+"alread registed. ", peerId)
		return nil, errors.New("already exsit peer node")
	}

	// connect rpc server
	t.mutex.Lock()
	defer t.mutex.Unlock()
	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.WithField("RpcTransporter", "transporter.RegistPeerNode").Error(goidForlog()+"err : ", err)
		return nil, err
	}

	// regist peer
	peerNode := &RaftPeerNode{
		id:        int(peerInfo.GetId()),
		address:   peerInfo.Address,
		requestor: &RpcRequestor{client},
	}

	t.peers[peerId] = peerInfo
	return peerNode, nil
}

func (t *RpcTransporter) RemovePeerNode(peerId int) {
	delete(t.peers, peerId)
}

func (t *RpcTransporter) Stop() {
	t.quitSinal <- true
	t.running = false
	t.wg.Wait()
}

func (t *RpcTransporter) ConnectToPeer(args *message.RegistPeer, reply *bool) error {
	log.WithField("RpcTransporter", "transporter.ConnectToPeer").Info(goidForlog())
	peerNode, err := t.RegistPeerNode(args)
	if peerNode != nil {
		t.handler.onConnectToPeer(peerNode)
	}

	*reply = (err == nil)
	return err
}

func (t *RpcTransporter) RequestVote(args *message.RequestVote, reply *message.RequestVoteReply) error {
	log.WithField("RpcTransporter", "transporter.RequestVote").Info(goidForlog())
	if t.handler == nil {
		return errors.New("invalid handler")
	}
	t.handler.onRequestVote(args, reply)
	return nil
}

func (t *RpcTransporter) AppendEntries(args *message.AppendEntries, reply *message.AppendEntriesReply) error {
	log.WithField("RpcTransporter", "transporter.AppendEntries").Info(goidForlog())
	if t.handler == nil {
		return errors.New("invalid handler")
	}
	t.handler.onAppendEntries(args, reply)
	return nil
}

func (t *RpcTransporter) ApplyEntry(args *message.ApplyEntry, reply *bool) error {
	log.WithField("RpcTransporter", "transporter.ApplyEntry").Info(goidForlog())
	if t.handler == nil {
		return errors.New("invalid handler")
	}
	t.handler.onApplyEntry(args)

	*reply = true
	return nil
}
