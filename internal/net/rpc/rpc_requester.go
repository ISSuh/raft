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
	"fmt"
	"math/rand"
	"net/rpc"
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"github.com/ISSuh/raft/internal/util"
	"google.golang.org/protobuf/proto"
)

type RpcRequester struct {
	client         *rpc.Client
	requestTimeout time.Duration
}

func NewNodeRequester(client *rpc.Client, timeout int) net.NodeRequester {
	requestTimeout := net.DefaultRequestTimneout
	if timeout > 0 {
		requestTimeout = time.Duration(timeout) * time.Millisecond
	}

	return &RpcRequester{
		client:         client,
		requestTimeout: requestTimeout,
	}
}

func NewClusterRequester(client *rpc.Client, timeout int) net.ClusterRequester {
	requestTimeout := net.DefaultRequestTimneout
	if timeout > 0 {
		requestTimeout = time.Duration(requestTimeout) * time.Millisecond
	}

	return &RpcRequester{
		client:         client,
		requestTimeout: requestTimeout,
	}
}

func (r *RpcRequester) CloseNodeRequester() {
	r.client.Close()
}

func (r *RpcRequester) CloseClusterRequester() {
	r.client.Close()
}

func (r *RpcRequester) HelthCheck() error {
	req := RpcRequest{
		Id:      rand.Uint32(),
		Type:    event.HealthCheck,
		Message: []byte{},
	}

	resp := RpcResponse{}

	return r.rpcCall(&req, &resp)
}

func (r *RpcRequester) NotifyNodeConnected(node *message.NodeMetadata) (bool, error) {
	data, err := proto.Marshal(node)
	if err != nil {
		return false, err
	}

	req, resp := r.makeRpcRequestResponse(event.NotifyNodeConnected, data)
	if err := r.rpcCall(&req, &resp); err != nil {
		return false, err
	}

	if req.Id != resp.Id {
		return false,
			fmt.Errorf("[RequestVote] not matched request, respnse id. [request id = %d, response id = %d]",
				req.Id, resp.Id,
			)
	}
	return util.BooleanByteSliceToBool(resp.Message), nil
}

func (r *RpcRequester) NotifyNodeDisconnected(node *message.NodeMetadata) error {
	data, err := proto.Marshal(node)
	if err != nil {
		return err
	}

	req, resp := r.makeRpcRequestResponse(event.NotifyNodeDisconnected, data)
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
		return err
	}
	return nil
}

func (r *RpcRequester) RequestVote(requestVoteMessage *message.RequestVote) (*message.RequestVoteReply, error) {
	data, err := proto.Marshal(requestVoteMessage)
	if err != nil {
		return nil, err
	}

	req, resp := r.makeRpcRequestResponse(event.ReqeustVote, data)
	if err := r.rpcCall(&req, &resp); err != nil {
		return nil, err
	}

	if req.Id != resp.Id {
		return nil,
			fmt.Errorf("[RpcRequestor.RequestVote] not matched request, respnse id. [request id = %d, response id = %d]",
				req.Id, resp.Id,
			)
	}

	reply := &message.RequestVoteReply{}
	if err := proto.Unmarshal(resp.Message, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (r *RpcRequester) AppendEntries(arg *message.AppendEntries) (*message.AppendEntriesReply, error) {
	data, err := proto.Marshal(arg)
	if err != nil {
		return nil, err
	}

	req, resp := r.makeRpcRequestResponse(event.AppendEntries, data)
	if err := r.rpcCall(&req, &resp); err != nil {
		return nil, err
	}

	if req.Id != resp.Id {
		return nil,
			fmt.Errorf("[RpcRequestor.AppendEntries] not matched request, respnse id. [request id = %d, response id = %d]",
				req.Id, resp.Id,
			)
	}

	reply := &message.AppendEntriesReply{}
	if err := proto.Unmarshal(resp.Message, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (r *RpcRequester) NotifyMeToCluster(arg *message.NodeMetadata) (*message.NodeMetadataesList, error) {
	data, err := proto.Marshal(arg)
	if err != nil {
		return nil, err
	}

	req, resp := r.makeRpcRequestResponse(event.NotifyMeToCluster, data)
	if err := r.rpcCall(&req, &resp); err != nil {
		return nil, err
	}

	reply := &message.NodeMetadataesList{}
	if err := proto.Unmarshal(resp.Message, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (r *RpcRequester) DisconnectNode(arg *message.NodeMetadata) error {
	data, err := proto.Marshal(arg)
	if err != nil {
		return err
	}

	req, resp := r.makeRpcRequestResponse(event.DeleteNode, data)
	if err := r.rpcCall(&req, &resp); err != nil {
		return err
	}
	return nil
}

func (r *RpcRequester) NodeList() (*message.NodeMetadataesList, error) {
	req, resp := r.makeRpcRequestResponse(event.NodeList, []byte{})
	if err := r.rpcCall(&req, &resp); err != nil {
		return nil, err
	}

	reply := &message.NodeMetadataesList{}
	if err := proto.Unmarshal(resp.Message, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (r *RpcRequester) makeRpcRequestResponse(messageType event.EventType, message []byte) (RpcRequest, RpcResponse) {
	req := RpcRequest{
		Id:      rand.Uint32(),
		Type:    messageType,
		Message: message,
	}

	resp := RpcResponse{
		Id:      0,
		Message: []byte{},
	}
	return req, resp
}

func (r *RpcRequester) rpcCall(req interface{}, resp interface{}) error {
	channelBufferLen := 1
	call := r.client.Go(RpcMethodHandle, req, resp, make(chan *rpc.Call, channelBufferLen))
	select {
	case <-time.After(r.requestTimeout):
	case resp := <-call.Done:
		if resp != nil && resp.Error != nil {
			return resp.Error
		}
	}
	return nil
}
