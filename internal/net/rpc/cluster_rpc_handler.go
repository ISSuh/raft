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
	"fmt"
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"github.com/ISSuh/raft/internal/util"
	"google.golang.org/protobuf/proto"
)

type ClusterRpcHandler struct {
	eventChannel chan event.Event
	eventTimeout time.Duration
}

func NewClusterRpcHandler(eventChannel chan event.Event, timeout int) RpcHandler {
	eventTimeout := net.DefaultRequestTimneout
	if timeout > 0 {
		eventTimeout = time.Duration(timeout) * time.Millisecond
	}

	return &ClusterRpcHandler{
		eventChannel: eventChannel,
		eventTimeout: eventTimeout,
	}
}

func (h *ClusterRpcHandler) Handle(req *RpcRequest, resp *RpcResponse) error {
	var err error
	switch req.Type {
	case event.NotifyMeToCluster:
		err = h.processNotifyMeToClusterEvent(req, resp)
	case event.DeleteNode:
		err = h.processDeleteNodeEvent(req, resp)
	case event.NodeList:
		err = h.processNodeListEvent(req, resp)
	default:
		err = fmt.Errorf("invalid event type. %d", req.Type)
	}
	return err
}

func (h *ClusterRpcHandler) processNotifyMeToClusterEvent(req *RpcRequest, resp *RpcResponse) error {
	message := &message.NodeMetadata{}
	err := proto.Unmarshal(req.Message, message)
	if err != nil {
		return fmt.Errorf("[ClusterRpcHandler.processNotifyMeToClusterEvent] invalid message. %v\n", req.Message)
	}

	peers, err := h.notifyMeToCluster(message)
	if err != nil {
		return err
	}

	resultMessage, err := proto.Marshal(peers)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = resultMessage
	return nil
}

func (h *ClusterRpcHandler) notifyMeToCluster(node *message.NodeMetadata) (*message.NodeMetadataesList, error) {
	fmt.Printf("[ClusterRpcHandler.notifyMe]\n")

	eventResult, err := h.notifyEvent(event.NotifyMeToCluster, node)
	if err != nil {
		return nil, err
	}

	if eventResult.Err != nil {
		return nil, eventResult.Err
	}

	result, ok := eventResult.Result.(*message.NodeMetadataesList)
	if !ok {
		return nil, fmt.Errorf("[ClusterRpcHandler.notifyMe] invalid event response. %v\n", eventResult)
	}
	return result, nil
}

func (h *ClusterRpcHandler) processDeleteNodeEvent(req *RpcRequest, resp *RpcResponse) error {
	message := &message.NodeMetadata{}
	err := proto.Unmarshal(req.Message, message)
	if err != nil {
		return fmt.Errorf("[ClusterRpcHandler.processDeleteNodeEvent] invalid message. %v\n", req.Message)
	}

	success, err := h.deleteNode(message)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = util.BooleanToByteSlice(success)
	return nil
}

func (h *ClusterRpcHandler) deleteNode(node *message.NodeMetadata) (bool, error) {
	fmt.Printf("[ClusterRpcHandler.DeleteNode]\n")

	eventResult, err := h.notifyEvent(event.DeleteNode, node)
	if err != nil {
		return false, err
	}

	if eventResult.Err != nil {
		return false, eventResult.Err
	}

	result, ok := eventResult.Result.(*bool)
	if !ok {
		return false, fmt.Errorf("[ClusterRpcHandler.deleteNode] invalid event response. %v\n", eventResult)
	}
	return *result, nil
}

func (h *ClusterRpcHandler) processNodeListEvent(req *RpcRequest, resp *RpcResponse) error {
	result, err := h.nodeList()
	if err != nil {
		return err
	}

	resultMessage, err := proto.Marshal(result)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = resultMessage
	return nil
}

func (h *ClusterRpcHandler) nodeList() (*message.NodeMetadataesList, error) {
	fmt.Printf("[ClusterRpcHandler.nodeList]\n")

	eventResult, err := h.notifyEvent(event.NodeList, nil)
	if err != nil {
		return nil, err
	}

	if eventResult.Err != nil {
		return nil, eventResult.Err
	}

	result, ok := eventResult.Result.(*message.NodeMetadataesList)
	if !ok {
		return nil, fmt.Errorf("[ClusterRpcHandler.nodeList] invalid event response. %v\n", eventResult)
	}
	return result, nil
}

func (h *ClusterRpcHandler) notifyEvent(eventType event.EventType, message interface{}) (*event.EventResult, error) {
	e := event.NewEvent(eventType, message)
	result, err := e.Notify(h.eventChannel, h.eventTimeout)
	if err != nil {
		return nil, err
	}
	return result, nil
}
