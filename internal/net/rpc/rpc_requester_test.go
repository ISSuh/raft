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
	"bytes"
	"context"
	"testing"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"github.com/ISSuh/raft/internal/util"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var defaultNodeConfig config.Config = config.Config{
	Raft: config.RaftConfig{
		Cluster: config.ClussterConfig{
			Address: config.Address{
				Ip:   "127.0.0.1",
				Port: 33121,
			},
		},
		Node: config.NodeConfig{
			Id: 0,
			Address: config.Address{
				Ip:   "127.0.0.1",
				Port: 33122,
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

func newTestRequesterForRequester(peerAddress config.Address) (net.NodeRequester, error) {
	testConfig := config.Config{
		Raft: config.RaftConfig{
			Node: config.NodeConfig{
				Id: 0,
				Address: config.Address{
					Ip:   "127.0.0.1",
					Port: util.RandRange(30000, 65536),
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
			Ip:   peerAddress.Ip,
			Port: int32(peerAddress.Port),
		},
	}
	return transporter.ConnectNode(node)
}

func runMockClusterForRequster(t *testing.T, c context.Context, config config.ClussterConfig) *RpcTransporter {
	h := &MockRpcHandler{}
	h.On("Handle", mock.Anything, mock.Anything).Return(nil)

	transporter := NewRpcTransporter(config.Address, config.Transport, h)
	err := transporter.Serve(c)
	require.NoError(t, err)
	return transporter
}

func runMockNodeForRequster(t *testing.T, c context.Context, config config.NodeConfig, h RpcHandler) *RpcTransporter {
	transporter := NewRpcTransporter(config.Address, config.Transport, h)
	err := transporter.Serve(c)
	require.NoError(t, err)
	return transporter
}

func TestHealthCheck(t *testing.T) {
	// given
	config := defaultNodeConfig
	config.Raft.Cluster.Address.Port = util.RandRange(30000, 65536)
	config.Raft.Node.Address.Port = util.RandRange(30000, 65536)

	mockHandler := &MockRpcHandler{}
	mockHandler.On("Handle", mock.Anything, mock.Anything).Return(nil)
	peer := runMockNodeForRequster(t, context.Background(), config.Raft.Node, mockHandler)

	requester, err := newTestRequesterForRequester(config.Raft.Node.Address)
	require.NoError(t, err)
	require.NotNil(t, requester)

	// when
	err = requester.HelthCheck()

	// then
	require.NoError(t, err)
	mockHandler.AssertExpectations(t)

	peer.StopAndWait()
}

func TestHealthCheckFail(t *testing.T) {
	// given
	config := defaultNodeConfig
	config.Raft.Cluster.Address.Port = util.RandRange(30000, 65536)
	config.Raft.Node.Address.Port = util.RandRange(30000, 65536)

	mockHandler := &MockRpcHandler{}
	mockHandler.On("Handle", mock.Anything, mock.Anything).Return(nil)
	peer := runMockNodeForRequster(t, context.Background(), config.Raft.Node, mockHandler)

	requester, err := newTestRequesterForRequester(config.Raft.Node.Address)
	require.NoError(t, err)
	require.NotNil(t, requester)

	peer.StopAndWait()

	// when
	err = requester.HelthCheck()

	// then
	require.Error(t, err)
}

func TestRequestVoteOkVote(t *testing.T) {
	// given
	config := defaultNodeConfig
	config.Raft.Cluster.Address.Port = util.RandRange(30000, 65536)
	config.Raft.Node.Address.Port = util.RandRange(30000, 65536)

	TestTerm := uint64(5)
	TestCandidateId := int32(1)
	m := &message.RequestVote{
		Term:        TestTerm,
		CandidateId: TestCandidateId,
	}

	p := &message.RequestVoteReply{
		Term:        TestTerm,
		VoteGranted: true,
	}

	requestId := uint32(0)
	matchRequst := func(req *RpcRequest) bool {
		requestId = req.Id
		if req.Type != event.ReqeustVote {
			return false
		}

		msg, err := proto.Marshal(m)
		if err != nil || !bytes.Equal(msg, req.Message) {
			return false
		}
		return true
	}

	matchResponse := func(resp *RpcResponse) bool {
		msg, err := proto.Marshal(p)
		if err != nil {
			return false
		}

		resp.Id = requestId
		resp.Message = msg
		return true
	}

	mockHandler := &MockRpcHandler{}
	mockHandler.On("Handle", mock.MatchedBy(matchRequst), mock.MatchedBy(matchResponse)).Return(nil)
	runMockNodeForRequster(t, context.Background(), config.Raft.Node, mockHandler)

	requester, err := newTestRequesterForRequester(config.Raft.Node.Address)
	require.NoError(t, err)
	require.NotNil(t, requester)

	// whern
	requestVoteReply, err := requester.RequestVote(m)

	// then
	require.Nil(t, err)
	mockHandler.AssertExpectations(t)
	require.Equal(t, p.Term, requestVoteReply.Term)
	require.True(t, requestVoteReply.VoteGranted)
}

func TestRequestVoteFalseVote(t *testing.T) {
	// given
	config := defaultNodeConfig
	config.Raft.Cluster.Address.Port = util.RandRange(30000, 65536)
	config.Raft.Node.Address.Port = util.RandRange(30000, 65536)

	TestTerm := uint64(5)
	TestCandidateId := int32(1)
	m := &message.RequestVote{
		Term:        TestTerm,
		CandidateId: TestCandidateId,
	}

	p := &message.RequestVoteReply{
		Term:        TestTerm + 1,
		VoteGranted: false,
	}

	requestId := uint32(0)
	matchRequst := func(req *RpcRequest) bool {
		requestId = req.Id
		if req.Type != event.ReqeustVote {
			return false
		}

		msg, err := proto.Marshal(m)
		if err != nil || !bytes.Equal(msg, req.Message) {
			return false
		}
		return true
	}

	matchResponse := func(resp *RpcResponse) bool {
		msg, err := proto.Marshal(p)
		if err != nil {
			return false
		}

		resp.Id = requestId
		resp.Message = msg
		return true
	}

	mockHandler := &MockRpcHandler{}
	mockHandler.On("Handle", mock.MatchedBy(matchRequst), mock.MatchedBy(matchResponse)).Return(nil)
	runMockNodeForRequster(t, context.Background(), config.Raft.Node, mockHandler)

	requester, err := newTestRequesterForRequester(config.Raft.Node.Address)
	require.NoError(t, err)
	require.NotNil(t, requester)

	// whern
	requestVoteReply, err := requester.RequestVote(m)

	// then
	require.Nil(t, err)
	mockHandler.AssertExpectations(t)
	require.Greater(t, requestVoteReply.Term, m.Term)
	require.Equal(t, p.Term, requestVoteReply.Term)
	require.False(t, requestVoteReply.VoteGranted)
}
