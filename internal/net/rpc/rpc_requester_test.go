/*
MIT License

# Copyright (c) 2024 ISSuh

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
	"os"
	"testing"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var testClusterConfig config.Config = config.Config{
	Raft: config.RaftConfig{
		Cluster: config.ClussterConfig{
			Address: config.Address{
				Ip:   "127.0.0.1",
				Port: 22450,
			},
		},
	},
}

var testNodeConfig config.Config = config.Config{
	Raft: config.RaftConfig{
		Node: config.NodeConfig{
			Id: 0,
			Address: config.Address{
				Ip:   "127.0.0.1",
				Port: 22451,
			},
		},
	},
}

var nodeTransporter *RpcTransporter = nil
var clusterTransporter *RpcTransporter = nil
var mockHandler *mockNodeRpcHandler = nil

type mockNodeRpcHandler struct {
	Called map[event.EventType]bool
}

func newMockHandler() *mockNodeRpcHandler {
	return &mockNodeRpcHandler{
		Called: make(map[event.EventType]bool),
	}
}

func (m *mockNodeRpcHandler) Clear() {
	m.Called = make(map[event.EventType]bool)
}

func (m *mockNodeRpcHandler) Handle(req *RpcRequest, resp *RpcResponse) error {
	var err error
	switch req.Type {
	case event.HealthCheck:
		err = m.HelthCheck(req, resp)
	case event.NotifyNodeConnected:
		err = m.processNotifyMeToNodeEvent(req, resp)
	case event.ReqeustVote:
		err = m.processRequestVoteEvent(req, resp)
	case event.AppendEntries:
		err = m.processAppendEntriesEvent(req, resp)
	case event.ApplyEntry:
		err = m.processApplyEntryEvent(req, resp)
	case event.NotifyMeToCluster:
		err = m.processNotifyMeToClusterEvent(req, resp)
	case event.DeleteNode:
		err = m.processDeleteNodeEvent(req, resp)
	case event.NodeList:
		err = m.processNodeListEvent(req, resp)
	}
	return err
}

func (m *mockNodeRpcHandler) HelthCheck(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.HealthCheck] = true

	resp.Id = req.Id
	resp.Message = []byte{byte(1)}
	return nil
}

func (m *mockNodeRpcHandler) processNotifyMeToNodeEvent(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.NotifyNodeConnected] = true
	return nil
}

func (m *mockNodeRpcHandler) processRequestVoteEvent(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.ReqeustVote] = true

	requestVote := &message.RequestVote{}
	err := proto.Unmarshal(req.Message, requestVote)
	if err != nil {
		return fmt.Errorf("[processRequestVoteEvent] invalid message. %v\n", req.Message)
	}

	requestVoteReply := &message.RequestVoteReply{
		Term:        requestVote.Term,
		VoteGranted: true,
	}
	resultMessage, err := proto.Marshal(requestVoteReply)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = resultMessage
	return nil
}

func (m *mockNodeRpcHandler) processAppendEntriesEvent(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.AppendEntries] = true
	return nil
}

func (m *mockNodeRpcHandler) processApplyEntryEvent(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.ApplyEntry] = true
	return nil
}

func (m *mockNodeRpcHandler) processNotifyMeToClusterEvent(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.NotifyMeToCluster] = true
	return nil
}

func (m *mockNodeRpcHandler) processDeleteNodeEvent(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.DeleteNode] = true
	return nil
}

func (m *mockNodeRpcHandler) processNodeListEvent(req *RpcRequest, resp *RpcResponse) error {
	m.Called[event.NodeList] = true
	return nil
}

func newTestRequester() (net.NodeRequester, error) {
	testConfig := config.Config{
		Raft: config.RaftConfig{
			Node: config.NodeConfig{
				Id: 0,
				Address: config.Address{
					Ip:   "127.0.0.1",
					Port: 33158,
				},
				Event: config.Event{
					Timeout: 1000,
				},
				Transport: config.Transport{
					RequestTimeout: 1000,
				},
			},
		},
	}
	transporter := NewRpcTransporter(testConfig.Raft.Node.Address, testConfig.Raft.Node.Transport, nil)

	node := &message.NodeMetadata{
		Id: 1,
		Address: &message.Address{
			Ip:   testNodeConfig.Raft.Node.Address.Ip,
			Port: int32(testNodeConfig.Raft.Node.Address.Port),
		},
	}
	return transporter.ConnectNode(node)
}

func TestMain(m *testing.M) {
	c, nodeCancel := context.WithCancel(context.Background())
	mockHandler = newMockHandler()

	nodeTransporter = NewRpcTransporter(
		testNodeConfig.Raft.Node.Address, testNodeConfig.Raft.Node.Transport, mockHandler,
	)
	if err := nodeTransporter.Serve(c); err != nil {
		os.Exit(0)
	}

	clusterTransporter = NewRpcTransporter(
		testClusterConfig.Raft.Cluster.Address, testClusterConfig.Raft.Cluster.Transport, mockHandler,
	)
	if err := clusterTransporter.Serve(c); err != nil {
		os.Exit(0)
	}

	exitVal := m.Run()

	nodeCancel()
	os.Exit(exitVal)
}

func TestHealthCheck(t *testing.T) {
	requester, err := newTestRequester()
	require.Nil(t, err)

	err = requester.HelthCheck()
	require.Nil(t, err)

	require.True(t, mockHandler.Called[event.HealthCheck])

	mockHandler.Clear()
}

func TestRequestVote(t *testing.T) {
	requester, err := newTestRequester()
	require.Nil(t, err)

	TestTerm := uint64(5)
	TestCandidateId := int32(1)
	message := &message.RequestVote{
		Term:        TestTerm,
		CandidateId: TestCandidateId,
	}

	requestVoteReply, err := requester.RequestVote(message)
	require.Nil(t, err)

	require.True(t, mockHandler.Called[event.ReqeustVote])
	require.Equal(t, requestVoteReply.Term, TestTerm)
	require.True(t, requestVoteReply.VoteGranted)

	mockHandler.Clear()
}
