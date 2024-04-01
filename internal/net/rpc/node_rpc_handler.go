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

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/util"
	"google.golang.org/protobuf/proto"
)

type NodeRpcHandler struct {
	eventChannel chan event.Event
}

func NewNodeRpcHandler(eventChannel chan event.Event) RpcHandler {
	return &NodeRpcHandler{
		eventChannel: eventChannel,
	}
}

func (h *NodeRpcHandler) Handle(req *RpcRequest, resp *RpcResponse) error {
	fmt.Printf("[NodeRpcHandler.Handle]\n")
	var err error
	switch req.Type {
	case event.HealthCheck:
		err = h.HelthCheck(req, resp)
	case event.NotifyMeToNode:
		err = h.procesNotifyMeToNodeEvent(req, resp)
	case event.ReqeustVote:
		err = h.processRequestVoteEvent(req, resp)
	case event.AppendEntries:
		err = h.processAppendEntriesEvent(req, resp)
	case event.ApplyEntry:
		err = h.processApplyEntryEvent(req, resp)
	default:
		err = fmt.Errorf("invalid event type. %d", req.Type)
	}
	return err
}

func (h *NodeRpcHandler) HelthCheck(req *RpcRequest, resp *RpcResponse) error {
	resp.Id = req.Id
	resp.Message = []byte{byte(1)}
	return nil
}

func (h *NodeRpcHandler) procesNotifyMeToNodeEvent(req *RpcRequest, resp *RpcResponse) error {
	message := &message.NodeMetadata{}
	err := proto.Unmarshal(req.Message, message)
	if err != nil {
		return fmt.Errorf("[ClusterRpcHandler.procesNotifyMeToNodeEvent] invalid message. %v\n", req.Message)
	}

	success, err := h.notifyMeToNode(message)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = []byte{util.BooleanToByte(success)}
	return nil
}

func (h *NodeRpcHandler) notifyMeToNode(node *message.NodeMetadata) (bool, error) {
	fmt.Printf("[NodeRpcHandler.RequestVote]\n")

	eventResult, err := h.notifyEvent(event.ReqeustVote, node)
	if err != nil {
		return false, err
	}

	result, ok := eventResult.Result.(*bool)
	if !ok {
		return false, fmt.Errorf("[NodeRpcHandler.RequestVote] invalid event response. %v\n", eventResult)
	}

	if eventResult.Err != nil {
		return false, eventResult.Err
	}
	return *result, eventResult.Err
}

func (h *NodeRpcHandler) processRequestVoteEvent(req *RpcRequest, resp *RpcResponse) error {
	message := &message.RequestVote{}
	err := proto.Unmarshal(req.Message, message)
	if err != nil {
		return fmt.Errorf("[NodeRpcHandler.processRequestVoteEvent] invalid message. %v\n", req.Message)
	}

	requestVoteReply, err := h.RequestVote(message)
	if err != nil {
		return err
	}

	resultMessage, err := proto.Marshal(requestVoteReply)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = resultMessage
	return nil
}

func (h *NodeRpcHandler) RequestVote(requestVoteMessage *message.RequestVote) (*message.RequestVoteReply, error) {
	fmt.Printf("[NodeRpcHandler.RequestVote]\n")

	eventResult, err := h.notifyEvent(event.ReqeustVote, requestVoteMessage)
	if err != nil {
		return nil, err
	}

	result, ok := eventResult.Result.(*message.RequestVoteReply)
	if !ok {
		return nil, fmt.Errorf("[NodeRpcHandler.RequestVote] invalid event response. %v\n", eventResult)
	}
	return result, eventResult.Err
}

func (h *NodeRpcHandler) processAppendEntriesEvent(req *RpcRequest, resp *RpcResponse) error {
	message := &message.AppendEntries{}
	err := proto.Unmarshal(req.Message, message)
	if err != nil {
		return fmt.Errorf("[NodeRpcHandler.processAppendEntriesEvent] invalid message. %v\n", req.Message)
	}

	appendEntriesReply, err := h.AppendEntries(message)
	if err != nil {
		return err
	}

	resultMessage, err := proto.Marshal(appendEntriesReply)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = resultMessage
	return nil
}

func (h *NodeRpcHandler) AppendEntries(appendEntriesMessage *message.AppendEntries) (*message.AppendEntriesReply, error) {
	fmt.Printf("[NodeRpcHandler.AppendEntries]\n")

	eventResult, err := h.notifyEvent(event.AppendEntries, appendEntriesMessage)
	if err != nil {
		return nil, err
	}

	result, ok := eventResult.Result.(*message.AppendEntriesReply)
	if !ok {
		return nil, fmt.Errorf("[NodeRpcHandler.AppendEntries] invalid event response. %v\n", eventResult)
	}
	return result, eventResult.Err
}

func (h *NodeRpcHandler) processApplyEntryEvent(req *RpcRequest, resp *RpcResponse) error {
	message := &message.ApplyEntry{}
	err := proto.Unmarshal(req.Message, message)
	if err != nil {
		return fmt.Errorf("[NodeRpcHandler.processApplyEntryEvent] invalid message. %v\n", req.Message)
	}

	success, err := h.ApplyEntry(message)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = []byte{util.BooleanToByte(success)}
	return nil
}

func (h *NodeRpcHandler) ApplyEntry(applyEntryMessage *message.ApplyEntry) (bool, error) {
	fmt.Printf("[NodeRpcHandler.ApplyEntry]\n")

	eventResult, err := h.notifyEvent(event.ReqeustVote, applyEntryMessage)
	if err != nil {
		return false, err
	}

	result, ok := eventResult.Result.(*bool)
	if !ok {
		return false, fmt.Errorf("[NodeRpcHandler.ApplyEntry] invalid event response. %v\n", eventResult)
	}

	if eventResult.Err != nil {
		return false, eventResult.Err
	}
	return *result, nil
}

func (h *NodeRpcHandler) notifyEvent(eventType event.EventType, message interface{}) (*event.EventResult, error) {
	e := event.NewEvent(eventType, message)
	result, err := e.Notify(h.eventChannel)
	if err != nil {
		return nil, err
	}
	return result, nil
}
