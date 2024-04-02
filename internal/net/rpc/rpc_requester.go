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
	"strconv"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"google.golang.org/protobuf/proto"
)

type RpcRequester struct {
	client *rpc.Client
}

func NewNodeRequester(client *rpc.Client) net.NodeRequester {
	return &RpcRequester{
		client: client,
	}
}

func NewClusterRequester(client *rpc.Client) net.ClusterRequester {
	return &RpcRequester{
		client: client,
	}
}

func (r *RpcRequester) HelthCheck() error {
	req := RpcRequest{
		Id:      rand.Uint32(),
		Type:    event.HealthCheck,
		Message: []byte{},
	}

	resp := RpcResponse{}
	return r.client.Call(RpcMethodHandle, &req, &resp)
}

func (r *RpcRequester) NotifyMeToPeerNode(myNode *message.NodeMetadata) (bool, error) {
	data, err := proto.Marshal(myNode)
	if err != nil {
		return false, err
	}

	req, resp := r.makeRpcRequestResponse(event.NotifyMeToNode, data)
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
		return false, err
	}

	if req.Id != resp.Id {
		return false,
			fmt.Errorf("[RpcRequestor.RequestVote] not matched request, respnse id. [request id = %d, response id = %d]",
				req.Id, resp.Id,
			)
	}

	reply, err := strconv.ParseBool(string(resp.Message))
	if err != nil {
		return false, err
	}
	return reply, nil
}

func (r *RpcRequester) Disconnect(myNode *message.NodeMetadata) error {
	data, err := proto.Marshal(myNode)
	if err != nil {
		r.client.Close()
		return err
	}

	req, resp := r.makeRpcRequestResponse(event.NotifyMeToNode, data)
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
		r.client.Close()
		return err
	}

	r.client.Close()
	return nil
}

func (r *RpcRequester) RequestVote(requestVoteMessage *message.RequestVote) (*message.RequestVoteReply, error) {
	data, err := proto.Marshal(requestVoteMessage)
	if err != nil {
		return nil, err
	}

	req, resp := r.makeRpcRequestResponse(event.ReqeustVote, data)
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
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

	req, resp := r.makeRpcRequestResponse(event.ReqeustVote, data)
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
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
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
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
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
		return err
	}

	_, err = strconv.ParseBool(string(resp.Message))
	if err != nil {
		return err
	}
	return nil
}

func (r *RpcRequester) NodeList() (*message.NodeMetadataesList, error) {
	req, resp := r.makeRpcRequestResponse(event.NodeList, []byte{})
	if err := r.client.Call(RpcMethodHandle, &req, &resp); err != nil {
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
