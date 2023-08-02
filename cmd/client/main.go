package main

import (
	"log"
	"net"
	"net/rpc"
	"os"

	"github.com/ISSuh/raft"
)

const (
	Localhost = "localhost"
	RpcMethod = "Raft.ApplyEntry"
)

func main() {
	log.Println("RAFT ")
	args := os.Args[1:]
	if len(args) < 1 {
		log.Println("invalid number of arguments")
		return
	}

	port := args[0]
	address := Localhost + ":" + port
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.Println("net.ResolveTCPAddr err : ", err)
		return
	}

	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.Println("rpc.Dial err : ", err)
		return
	}

	applyEntry := raft.ApplyEntry{
		Command: []byte("TEST"),
	}

	var applyEntryReply raft.ApplyEntryReply
	err = client.Call(RpcMethod, applyEntry, &applyEntryReply)
	if err != nil {
		log.Println("client.Call err : ", err)
	}

	log.Println("respone: ", applyEntryReply.Success)
}
