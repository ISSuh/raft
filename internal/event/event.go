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

package event

import (
	"fmt"
	"time"
)

const (
	DefaultEventTimeout = 3 * time.Second
)

type EventResult struct {
	Err    error
	Result interface{}
}

type Event struct {
	Type               EventType
	Message            interface{}
	Timestamp          time.Time
	EventResultChannel chan *EventResult
}

func NewEvent(eventType EventType, message interface{}) Event {
	return Event{
		Type:               eventType,
		Message:            message,
		Timestamp:          time.Now(),
		EventResultChannel: make(chan *EventResult),
	}
}

func (e Event) String() string {
	return fmt.Sprintf("Event{Type:%s, Message:%+v, Timestamp:%s}", e.Type, e.Message, e.Timestamp)
}

func (e *Event) Notify(eventChannel chan Event, timeout time.Duration) (*EventResult, error) {
	eventNotifyTimeoutChan := time.After(timeout)
	select {
	case eventChannel <- *e:
	case <-eventNotifyTimeoutChan:
		return nil, fmt.Errorf("[Notify] %s evnet push timeout.", e.Type)
	}

	eventResultTimeoutChan := time.After(timeout)
	select {
	case result := <-e.EventResultChannel:
		return result, nil
	case <-eventResultTimeoutChan:
		return nil, fmt.Errorf("[Notify] %s result evnet timeout.", e.Type)
	}
}

func (e *Event) Reply(result *EventResult) error {
	select {
	case e.EventResultChannel <- result:
	default:
		return fmt.Errorf("[Reply] evnet result channel already closed")
	}
	return nil
}
