package main

import (
	"fmt"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/event"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
)

func main() {
	eventChan := make(chan event.Event)

	config := config.Config{
		Raft: config.RaftConfig{
			Server: config.ServerConfig{
				Id: 0,
				Address: config.Address{
					Ip:   "0.0.0.0",
					Port: 33113,
				},
			},
		},
	}

	t := net.NewRpcTransporter(config, eventChan)

	node := message.NodeMetadata{
		Id: 1,
		Address: &message.Address{
			Ip:   "127.0.0.1",
			Port: 33112,
		},
	}

	requestor, err := t.ConnectPeerNode(node)
	if err != nil {
		return
	}

	result := false
	if err := requestor.HelthCheck(&result); err != nil {
		fmt.Println(err.Error())
		return
	}

	for i := 0; i < 100; i++ {
		var args message.RequestVote
		var reply message.RequestVoteReply
		if err := requestor.RequestVote(&args, &reply); err != nil {
			fmt.Println(err.Error())
		}
		fmt.Printf("reply : %+v\n", reply)
	}
}
