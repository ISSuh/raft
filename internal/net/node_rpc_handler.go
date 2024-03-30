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

package net

import (
	"fmt"
	"time"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
)

type RpcHandler struct {
	eventChannel chan event.Event
}

func NewRpcHandler(eventChannel chan event.Event) *RpcHandler {
	return &RpcHandler{
		eventChannel: eventChannel,
	}
}

func (h *RpcHandler) HelthCheck(args *bool, reply *bool) error {
	*reply = true
	return nil
}

func (h *RpcHandler) RequestVote(args *message.RequestVote, reply *message.RequestVoteReply) error {
	fmt.Printf("[RpcHandler.RequestVote]\n")

	eventResult, err := h.sendEvent(event.ReqeustVote, args)
	if err != nil {
		return err
	}

	result, ok := eventResult.(*message.RequestVoteReply)
	if !ok {
		return fmt.Errorf("[RpcHandler.RequestVote] invalid event response. %v\n", eventResult)
	}

	reply.Term = result.Term
	reply.VoteGranted = result.VoteGranted
	return nil
}

func (h *RpcHandler) AppendEntries(args *message.AppendEntries, reply *message.AppendEntriesReply) error {
	fmt.Printf("[RpcHandler.AppendEntries]\n")

	eventResult, err := h.sendEvent(event.AppendEntries, args)
	if err != nil {
		return err
	}

	result, ok := eventResult.(*message.AppendEntriesReply)
	if !ok {
		return fmt.Errorf("[RpcHandler.AppendEntries] invalid event response. %v\n", eventResult)
	}

	reply.Term = result.Term
	reply.PeerId = result.PeerId
	reply.Success = result.Success
	reply.ConflictTerm = result.ConflictTerm
	reply.ConflictIndex = result.ConflictIndex
	return nil
}

func (h *RpcHandler) ApplyEntry(args *message.ApplyEntry, reply *bool) error {
	fmt.Printf("[RpcHandler.ApplyEntry]\n")

	eventResult, err := h.sendEvent(event.ReqeustVote, args)
	if err != nil {
		return err
	}

	result, ok := eventResult.(*bool)
	if !ok {
		return fmt.Errorf("[RpcHandler.ApplyEntry] invalid event response. %v\n", eventResult)
	}

	*reply = *result
	return nil
}

func (h *RpcHandler) sendEvent(eventType event.EventType, message interface{}) (interface{}, error) {
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
		return nil, fmt.Errorf("[RpcHandler.sendEvent] %s evnet timeout.", eventType)
	}
}
