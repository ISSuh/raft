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

package node

import (
	"context"
	"errors"
	"testing"

	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFollowerWork(t *testing.T) {
	meta := &message.NodeMetadata{
		Id: 0,
	}

	// given
	n := NewNode(meta)
	q := make(chan struct{})
	p := &event.MockEventProcessor{}
	p.On("WaitUntilEmit").Return(
		func() <-chan event.Event {
			return make(chan event.Event)
		},
	)

	woker := NewFollowerStateWorker(n, p, q)

	// when
	woker.Work(context.Background())

	// then
	require.Equal(t, CandidateState, n.currentState())
}

func TestFollowerWorkStop(t *testing.T) {
	meta := &message.NodeMetadata{
		Id: 0,
	}

	t.Run("quit", func(t *testing.T) {
		// given
		n := NewNode(meta)
		q := make(chan struct{})
		p := &event.MockEventProcessor{}
		p.On("WaitUntilEmit").Return(
			func() <-chan event.Event {
				return make(chan event.Event)
			},
		)

		woker := NewFollowerStateWorker(n, p, q)

		go func() {
			q <- struct{}{}
		}()

		// when
		woker.Work(context.Background())

		// then
		require.Equal(t, StopState, n.currentState())
	})

	t.Run("context canel", func(t *testing.T) {
		// given
		n := NewNode(meta)
		q := make(chan struct{})
		p := &event.MockEventProcessor{}
		p.On("WaitUntilEmit").Return(
			func() <-chan event.Event {
				return make(chan event.Event)
			},
		)

		woker := NewFollowerStateWorker(n, p, q)

		c, candel := context.WithCancel(context.Background())

		go func() {
			candel()
		}()

		// when
		woker.Work(c)

		// then
		require.Equal(t, StopState, n.currentState())
	})
}

func TestFollowerWorkEvent(t *testing.T) {
	meta := &message.NodeMetadata{
		Id: 0,
	}

	ch := make(chan event.Event)

	p := &event.MockEventProcessor{}
	p.On("WaitUntilEmit").Return(
		func() <-chan event.Event {
			return ch
		},
	)

	p.On(
		"ProcessEvent",
		mock.Anything).Return(
		func(e event.Event) (interface{}, error) {
			switch e.Type {
			case event.ReqeustVote:
				fallthrough
			case event.AppendEntries:
				return nil, nil
			}
			return nil, errors.New("error")
		},
	)

	t.Run("event requestVote", func(t *testing.T) {
		// given
		n := NewNode(meta)
		q := make(chan struct{})
		woker := NewFollowerStateWorker(n, p, q)

		go func(t *testing.T) {
			testEvent := event.NewEvent(event.ReqeustVote, nil)
			ch <- testEvent

			result := <-testEvent.EventResultChannel
			require.NoError(t, result.Err)
		}(t)

		// when
		woker.Work(context.Background())

		// then
		require.Equal(t, CandidateState, n.currentState())
	})

	t.Run("event appendEntries", func(t *testing.T) {
		// given
		n := NewNode(meta)
		q := make(chan struct{})
		woker := NewFollowerStateWorker(n, p, q)

		go func(t *testing.T) {
			testEvent := event.NewEvent(event.AppendEntries, nil)
			ch <- testEvent

			result := <-testEvent.EventResultChannel
			require.NoError(t, result.Err)
		}(t)

		// when
		woker.Work(context.Background())

		// then
		require.Equal(t, CandidateState, n.currentState())
	})
}

func TestFollowerWorkFail(t *testing.T) {
	meta := &message.NodeMetadata{
		Id: 0,
	}

	t.Run("invalid state", func(t *testing.T) {
		// given
		n := NewNode(meta)
		q := make(chan struct{})
		woker := NewFollowerStateWorker(n, nil, q)

		// when
		n.setState(LeaderState)
		woker.Work(context.Background())

		// then
		require.Equal(t, LeaderState, n.currentState())
	})

	t.Run("invalid state before timer", func(t *testing.T) {
		// given
		ch := make(chan event.Event)
		n := NewNode(meta)
		q := make(chan struct{})
		p := &event.MockEventProcessor{}
		p.On("WaitUntilEmit").Return(
			func() <-chan event.Event {
				return ch
			},
		)

		p.On("ProcessEvent", mock.Anything).Return(nil, nil)

		woker := NewFollowerStateWorker(n, p, q)

		// when
		go func() {
			testEvent := event.NewEvent(event.ReqeustVote, nil)
			ch <- testEvent

			n.setState(LeaderState)

			<-testEvent.EventResultChannel
		}()

		woker.Work(context.Background())

		// then
		require.Equal(t, LeaderState, n.currentState())
	})
}

func TestFollowerWorkEventFail(t *testing.T) {
	meta := &message.NodeMetadata{
		Id: 0,
	}

	ch := make(chan event.Event)

	p := &event.MockEventProcessor{}
	p.On("WaitUntilEmit").Return(
		func() <-chan event.Event {
			return ch
		},
	)

	p.On(
		"ProcessEvent",
		mock.Anything).Return(
		func(e event.Event) (interface{}, error) {
			switch e.Type {
			case event.ReqeustVote:
				fallthrough
			case event.AppendEntries:
				return nil, nil
			}
			return nil, errors.New("error")
		},
	)

	t.Run("invalid event", func(t *testing.T) {
		// given
		n := NewNode(meta)
		q := make(chan struct{})
		woker := NewFollowerStateWorker(n, p, q)

		go func(t *testing.T) {
			testEvent := event.NewEvent(event.DeleteNode, nil)
			ch <- testEvent

			result := <-testEvent.EventResultChannel
			require.Error(t, result.Err)
		}(t)

		// when
		woker.Work(context.Background())

		// then
		require.Equal(t, CandidateState, n.currentState())
	})
}
