package raft

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
)

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term     int
	LeaderId int

	PrevLogIndex int
	PrevLogTerm  int
	// Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool

	// Faster conflict resolution optimization (described near the end of section
	// 5.3 in the paper.)
	ConflictIndex int
	ConflictTerm  int
}

type RPCProxy struct {
	node *RaftNode
}

func (rpp *RPCProxy) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	// return rpp.cm.RequestVote(args, reply)
	return nil
}

func (rpp *RPCProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	// return rpp.cm.AppendEntries(args, reply)
	return nil
}

type PeerInfo struct {
	id   int
	port string
}

type NetworkService struct {
	serverId int

	node        *RaftNode
	rpcServer   *rpc.Server
	listener    net.Listener
	rpcProxy    *RPCProxy
	peerClients map[int]*rpc.Client

	mutex sync.Mutex
	wg    sync.WaitGroup

	ready <-chan interface{}
	quit  chan interface{}
}

func NewNetworkService(node *RaftNode) *NetworkService {
	return &NetworkService{
		node: node,
	}
}

func (service *NetworkService) Serve(port string) {
	service.mutex.Lock()

	service.rpcServer = rpc.NewServer()
	service.rpcProxy = &RPCProxy{node: service.node}
	service.rpcServer.RegisterName("RaftNode", service.rpcProxy)

	var err error
	service.listener, err = net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[%v] listening at %s", service.serverId, service.listener.Addr())
	service.mutex.Unlock()

	service.wg.Add(1)
	go func() {
		defer service.wg.Done()

		for {
			conn, err := service.listener.Accept()
			if err != nil {
				select {
				case <-service.quit:
					return
				default:
					log.Fatal("accept error:", err)
				}
			}
			service.wg.Add(1)
			go func() {
				service.rpcServer.ServeConn(conn)
				service.wg.Done()
			}()
		}
	}()
}

func (service *NetworkService) ConnectToPeer(peerId int, addr net.Addr) error {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if service.peerClients[peerId] == nil {
		client, err := rpc.Dial(addr.Network(), addr.String())
		if err != nil {
			return err
		}
		service.peerClients[peerId] = client
	}
	return nil
}

func (service *NetworkService) Call(id int, serviceMethod string, args interface{}, reply interface{}) error {
	service.mutex.Lock()
	peer := service.peerClients[id]
	service.mutex.Unlock()

	if peer == nil {
		return fmt.Errorf("call client %d after it's closed", id)
	} else {
		return peer.Call(serviceMethod, args, reply)
	}
}
