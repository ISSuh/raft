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
	"math/rand"
	"net/rpc"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
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
		Message: false,
	}

	resp := RpcResponse{}
	return r.client.Call(RpcMethodHandle, &req, &resp)
}

func (r *RpcRequestor) ConnectToPeer(arg *message.NodeMetadata, reply *bool) error {
	return r.client.Call(RpcMethodConnectToPeer, arg, reply)
}

func (r *RpcRequestor) RequestVote(arg *message.RequestVote, reply *message.RequestVoteReply) error {
	req := RpcRequest{
		Id:      rand.Uint32(),
		Type:    event.HealthCheck,
		Message: arg,
	}

	resp := RpcResponse{}
	return r.client.Call(RpcMethodHandle, &req, &resp)
}

func (r *RpcRequestor) AppendEntries(arg *message.AppendEntries, reply *message.AppendEntriesReply) error {
	return r.client.Call(RpcMethodAppendEntries, arg, reply)
}
