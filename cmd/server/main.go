package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/ISSuh/raft"
	"github.com/ISSuh/raft/internal/logger"
)

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		logger.Info("invalid number of arguments\n")
		return
	}

	configPath := args[0]
	node, err := raft.NewRaftNode(configPath)
	if err != nil {
		logger.Info("%s\n", err.Error())
		return
	}

	c, candel := context.WithCancel(context.Background())
	err = node.Serve(c)
	if err != nil {
		logger.Info("%s\n", err.Error())
		return
	}

	count := 0
	logPrefix := "log_0"
	for {
		log := logPrefix + strconv.Itoa(count)
		node.AppendLog([]byte(log))

		count++
		time.Sleep(1 * time.Second)
	}

	candel()
}
