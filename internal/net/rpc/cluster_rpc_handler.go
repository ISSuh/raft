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

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
)

type ClusterRpcHandler struct {
	eventChannel chan<- event.Event
}

func NewClusterRpcHandler(eventChannel chan event.Event) RpcHandler {
	return &ClusterRpcHandler{
		eventChannel: eventChannel,
	}
}

func (h *ClusterRpcHandler) Handle(req *RpcRequest, resp *RpcResponse) error {
	var err error
	switch req.Type {
	case event.ConnectNode:
		err = h.processConnectNodeEvent(req, resp)
	case event.DeleteNode:
		err = h.processDeleteNodeEvent(req, resp)
	default:
		err = fmt.Errorf("invalid event type. %d", req.Type)
	}
	return err
}

func (h *ClusterRpcHandler) processConnectNodeEvent(req *RpcRequest, resp *RpcResponse) error {
	node, ok := req.Message.(*message.NodeMetadata)
	if !ok {
		return fmt.Errorf("[ClusterRpcHandler.processConnectNodeEvent] invalid message. %v\n", req.Message)
	}

	peers, err := h.connectNode(node)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = peers
	return nil
}

func (h *ClusterRpcHandler) connectNode(node *message.NodeMetadata) ([]*message.NodeMetadata, error) {
	fmt.Printf("[ClusterRpcHandler.ApplyEntry]\n")

	eventResult, err := h.notifyEvent(event.ConnectNode, node)
	if err != nil {
		return nil, err
	}

	if eventResult.Err != nil {
		return nil, eventResult.Err
	}

	result, ok := eventResult.Result.(*[]*message.NodeMetadata)
	if !ok {
		return nil, fmt.Errorf("[ClusterRpcHandler.ApplyEntry] invalid event response. %v\n", eventResult)
	}
	return *result, nil
}

func (h *ClusterRpcHandler) processDeleteNodeEvent(req *RpcRequest, resp *RpcResponse) error {
	node, ok := req.Message.(*message.NodeMetadata)
	if !ok {
		return fmt.Errorf("[ClusterRpcHandler.processDeleteNodeEvent] invalid message. %v\n", req.Message)
	}

	result, err := h.deleteNode(node)
	if err != nil {
		return err
	}

	resp.Id = req.Id
	resp.Message = result
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
		return false, fmt.Errorf("[ClusterRpcHandler.ApplyEntry] invalid event response. %v\n", eventResult)
	}
	return *result, nil
}

func (h *ClusterRpcHandler) notifyEvent(eventType event.EventType, message interface{}) (*event.EventResult, error) {
	e := event.NewEvent(eventType, message)
	result, err := e.Notify(h.eventChannel)
	if err != nil {
		return nil, err
	}
	return result, nil
}
