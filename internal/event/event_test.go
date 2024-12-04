package event

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNotify_Success(t *testing.T) {
	eventChannel := make(chan Event)
	eventType := ReqeustVote
	message := "TestMessage"
	event := NewEvent(eventType, message)

	go func() {
		select {
		case e := <-eventChannel:
			require.Equal(t, eventType, e.Type)
			require.Equal(t, message, e.Message)

			e.Reply(&EventResult{Result: "Success"})
		case <-time.After(DefaultEventTimeout):
			t.Error("event not received in time")
		}
	}()

	result, err := event.Notify(eventChannel, DefaultEventTimeout)
	require.NoError(t, err)
	require.Equal(t, "Success", result.Result)
}

func TestNotify_Timeout(t *testing.T) {
	eventChannel := make(chan Event)
	eventType := AppendEntries
	message := "TestMessage"
	event := NewEvent(eventType, message)

	_, err := event.Notify(eventChannel, 1*time.Millisecond)
	require.Error(t, err)
}

func TestReply_Success(t *testing.T) {
	eventType := ReqeustVote
	message := "TestMessage"
	event := NewEvent(eventType, message)
	result := &EventResult{Result: "Success"}

	go func() {
		select {
		case res := <-event.EventResultChannel:
			require.Equal(t, result.Result, res.Result)
		case <-time.After(DefaultEventTimeout):
			t.Error("result not received in time")
		}
	}()

	time.Sleep(1 * time.Second)

	err := event.Reply(result)
	require.NoError(t, err)
}

func TestReply_ChannelClosed(t *testing.T) {
	eventType := ReqeustVote
	message := "TestMessage"
	event := NewEvent(eventType, message)
	result := &EventResult{Result: "Success"}

	close(event.EventResultChannel)

	err := event.Reply(result)
	require.Error(t, err)
}
