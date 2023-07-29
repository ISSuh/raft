package raft

import (
	"net/rpc"

	log "github.com/sirupsen/logrus"
)

type RaftPeerNode struct {
	id      int
	address string
	client  *rpc.Client
}

func (peer *RaftPeerNode) RequestVote(args interface{}, reply interface{}) error {
	log.WithField("peer", "RaftPeerNode.RequestVote").Info(goidForlog())
	method := "Raft.RequestVote"
	return peer.client.Call(method, args, reply)
}

func (peer *RaftPeerNode) AppendEntries(args interface{}, reply interface{}) error {
	log.WithField("peer", "RaftPeerNode.AppendEntries").Info(goidForlog())
	method := "Raft.AppendEntries"
	return peer.client.Call(method, args, reply)
}
