package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/ISSuh/raft"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		log.Printf("invalid number of arguments\n")
		return
	}

	configPath := args[0]
	cluster, err := raft.NewCluster(configPath)
	if err != nil {
		log.Printf("%s\n", err.Error())
		return
	}

	c, candel := context.WithCancel(context.Background())
	err = cluster.Serve(c)
	if err != nil {
		log.Printf("%s\n", err.Error())
		return
	}

	for {
		time.Sleep(1 * time.Second)
	}

	candel()
}
