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
	"google.golang.org/protobuf/proto"
)

const (
	RpcMethodHandle        = "Raft.Handle"
	RpcMethodHelthCheck    = "Raft.HelthCheck"
	RpcMethodConnectToPeer = "Raft.ConnectToPeer"
	RpcMethodRequestVote   = "Raft.RequestVote"
	RpcMethodAppendEntries = "Raft.AppendEntries"
)

type RpcRequestor struct {
	client *rpc.Client
}

func (r *RpcRequestor) HelthCheck() error {
	req := RpcRequest{
		Id:      rand.Uint32(),
		Type:    event.HealthCheck,
		Message: []byte{},
	}

	resp := RpcResponse{}
	return r.client.Call(RpcMethodHandle, &req, &resp)
}

func (r *RpcRequestor) ConnectToPeer(arg *message.NodeMetadata) (bool, error) {
	data, err := proto.Marshal(arg)
	if err != nil {
		return false, err
	}

	req, resp := r.makeRpcRequestResponse(event.ReqeustVote, data)
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

func (r *RpcRequestor) RequestVote(arg *message.RequestVote) (*message.RequestVoteReply, error) {
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

func (r *RpcRequestor) AppendEntries(arg *message.AppendEntries) (*message.AppendEntriesReply, error) {
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
			fmt.Errorf("[RpcRequestor.RequestVote] not matched request, respnse id. [request id = %d, response id = %d]",
				req.Id, resp.Id,
			)
	}

	reply := &message.AppendEntriesReply{}
	if err := proto.Unmarshal(resp.Message, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

func (r *RpcRequestor) makeRpcRequestResponse(messageType event.EventType, message []byte) (RpcRequest, RpcResponse) {
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
