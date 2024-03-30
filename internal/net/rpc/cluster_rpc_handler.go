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
)

type ClusterRpcHandler struct {
	eventChannel chan<- event.Event
}

func NewClusterRpcHandler(eventChannel chan event.Event) *ClusterRpcHandler {
	return &ClusterRpcHandler{
		eventChannel: eventChannel,
	}
}

func (h *ClusterRpcHandler) ConnectNode(args *message.NodeMetadata, reply *[]*message.NodeMetadata) error {
	fmt.Printf("[ClusterRpcHandler.ApplyEntry]\n")

	eventResult, err := h.notifyEvent(event.ConnectNode, args)
	if err != nil {
		return err
	}

	result, ok := eventResult.(*[]*message.NodeMetadata)
	if !ok {
		return fmt.Errorf("[ClusterRpcHandler.ApplyEntry] invalid event response. %v\n", eventResult)
	}

	*reply = *result
	return nil
}

func (h *ClusterRpcHandler) DeleteNode(args *message.NodeMetadata, reply *bool) error {
	fmt.Printf("[ClusterRpcHandler.DeleteNode]\n")

	eventResult, err := h.notifyEvent(event.DeleteNode, args)
	if err != nil {
		return err
	}

	result, ok := eventResult.(*bool)
	if !ok {
		return fmt.Errorf("[ClusterRpcHandler.ApplyEntry] invalid event response. %v\n", eventResult)
	}

	*reply = *result
	return nil
}

func (h *ClusterRpcHandler) notifyEvent(eventType event.EventType, message interface{}) (interface{}, error) {
	rsultChannel := make(chan interface{})
	e := event.Event{
		Type:        eventType,
		Message:     message,
		Timestamp:   time.Now(),
		EventResult: rsultChannel,
	}

	h.eventChannel <- e

	timeoutChan := time.After(1 * time.Second)
	select {
	case result := <-rsultChannel:
		return result, nil
	case <-timeoutChan:
		return nil, fmt.Errorf("[ClusterRpcHandler.notifyEvent] %s evnet timeout.", eventType)
	}
}
