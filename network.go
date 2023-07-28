package raft

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
)

type PeerNodeInfo struct {
	Id      int    `json:"id"`
	Address string `json:"address"`
}

type RegistPeerNodeReply struct {
	Regist bool
}

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
	node           *RaftNode
	networkService *NetworkService
}

func (proxy *RPCProxy) RegistPeerNode(args PeerNodeInfo, reply *RegistPeerNodeReply) error {
	log.Println("[RegistPeerNode]")
	err := proxy.networkService.ConnectToPeer(PeerNodeInfo{
		Id:      args.Id,
		Address: args.Address,
	})

	reply.Regist = (err == nil)
	return err
}

func (proxy *RPCProxy) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	log.Println("[RequestVote]")

	// return rpp.cm.RequestVote(args, reply)
	return nil
}

func (proxy *RPCProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	log.Println("[RequestVote]")
	// return rpp.cm.AppendEntries(args, reply)
	return nil
}

type NetworkService struct {
	id      int
	address string

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

func NewNetworkService(id int, node *RaftNode) *NetworkService {
	return &NetworkService{
		id:          id,
		node:        node,
		peerClients: make(map[int]*rpc.Client),
	}
}

func (service *NetworkService) Serve(address string, peers []PeerNodeInfo) {
	service.mutex.Lock()

	service.address = address
	service.rpcServer = rpc.NewServer()
	service.rpcProxy = &RPCProxy{node: service.node, networkService: service}
	service.rpcServer.Register(service.rpcProxy)

	var err error
	service.listener, err = net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("[Serve]listening at ", service.listener.Addr())
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

			log.Println("[Serve] connected : ", conn.RemoteAddr().String())
			service.wg.Add(1)
			go func() {
				service.rpcServer.ServeConn(conn)
				service.wg.Done()
			}()
		}
	}()
}

func (service *NetworkService) GetListenAddr() net.Addr {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	return service.listener.Addr()
}

func (service *NetworkService) ConnectToPeer(peer PeerNodeInfo) error {
	log.Println("[ConnectToPeer] - peer : ", peer)
	peerId := peer.Id
	addr, err := net.ResolveTCPAddr("tcp", peer.Address)
	if err != nil {
		log.Println("[ConnectToPeer] err : ", err)
		return err
	}

	if service.peerClients[peerId] != nil {
		log.Println("[ConnectToPeer] alread registed. ", peer.Id)
		return nil
	}

	service.mutex.Lock()
	defer service.mutex.Unlock()
	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.Println("[ConnectToPeer] err : ", err)
		return err
	}
	service.peerClients[peerId] = client

	myInfo := PeerNodeInfo{
		Id:      service.id,
		Address: service.address,
	}

	var reply RegistPeerNodeReply
	err = client.Call("RPCProxy.RegistPeerNode", myInfo, &reply)
	if err != nil {
		log.Println("[ConnectToPeer] ", err)
	}

	log.Println("[ConnectToPeer] result : ", reply.Regist)
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
