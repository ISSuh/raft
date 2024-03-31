package main

import (
	"context"
	"fmt"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/net/rpc"
)

func main() {
	eventChan := make(chan event.Event)

	config := config.Config{
		Raft: config.RaftConfig{
			Server: config.ServerConfig{
				Id: 0,
				Address: config.Address{
					Ip:   "0.0.0.0",
					Port: 33112,
				},
			},
		},
	}

	h := rpc.NewNodeRpcHandler(eventChan)
	t := rpc.NewRpcTransporter(config, h)

	q := make(chan interface{})
	go func(eventChan <-chan event.Event, q <-chan interface{}) {
		for {
			select {
			case <-q:
				return
			case e := <-eventChan:
				fmt.Printf("[TEST] event : %s\n", e)
				e.EventResultChannel <- event.EventResult{
					Err:    nil,
					Result: true,
				}
			}

		}
	}(eventChan, q)

	c, cancel := context.WithCancel(context.Background())
	if err := t.Serve(c); err != nil {
		return
	}

	q <- true
	cancel()
	t.StopAndWait()
}
