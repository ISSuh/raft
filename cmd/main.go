package main

import (
	"log"

	"github.com/ISSuh/raft"
)

func main() {
	log.Println("RAFT")

	service := raft.NewRaftService()
	service.Run()
}
