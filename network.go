package raft

import (
	"fmt"
	"net"
	"net/rpc"
	"sync"

	log "github.com/sirupsen/logrus"
)

type PeerNodeInfo struct {
	Id      int    `json:"id"`
	Address string `json:"address"`
}

type RegistPeerNodeReply struct {
	Regist bool
}

type RequestVoteArgs struct {
	Term        uint64
	CandidateId int
	// LastLogIndex int
	// LastLogTerm  int
}

type RequestVoteReply struct {
	Term        uint64
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term     uint64
	LeaderId int

	// PrevLogIndex int
	// PrevLogTerm  int
	// Entries      []LogEntry
	// LeaderCommit int
}

type AppendEntriesReply struct {
	Term    uint64
	Success bool

	// Faster conflict resolution optimization (described near the end of section
	// 5.3 in the paper.)
	// ConflictIndex int
	// ConflictTerm  int
}

type RPCProxy struct {
	node           *RaftNode
	networkService *NetworkService
}

func (proxy *RPCProxy) RegistPeerNode(args PeerNodeInfo, reply *RegistPeerNodeReply) error {
	log.WithField("network", "network.RegistPeerNode").Info(goidForlog())
	err := proxy.networkService.ConnectToPeer(PeerNodeInfo{
		Id:      args.Id,
		Address: args.Address,
	})

	reply.Regist = (err == nil)
	return err
}

func (proxy *RPCProxy) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	log.WithField("network", "network.RequestVote").Info(goidForlog())

	// return rpp.cm.RequestVote(args, reply)
	return nil
}

func (proxy *RPCProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	log.WithField("network", "network.AppendEntries").Info(goidForlog())
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
	service.rpcServer.RegisterName("Raft", service.rpcProxy)

	var err error
	service.listener, err = net.Listen("tcp", address)
	if err != nil {
		log.WithField("network", "network.Serve").Fatal(err)
	}

	log.WithField("network", "network.Serve").Info(goidForlog()+"listening at ", service.listener.Addr())
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
					log.WithField("network", "network.Serve").Fatal(goidForlog()+"accept error:", err)
				}
			}

			log.WithField("network", "network.Serve").Info(goidForlog()+"connected : ", conn.RemoteAddr().String())
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
	log.WithField("network", "network.ConnectToPeer").Info(goidForlog()+"peer : ", peer)
	peerId := peer.Id
	addr, err := net.ResolveTCPAddr("tcp", peer.Address)
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	if service.peerClients[peerId] != nil {
		log.WithField("network", "network.ConnectToPeer").Warn(goidForlog()+"alread registed. ", peer.Id)
		return nil
	}

	service.mutex.Lock()
	defer service.mutex.Unlock()
	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}
	service.peerClients[peerId] = client

	myInfo := PeerNodeInfo{
		Id:      service.id,
		Address: service.address,
	}

	// notify peer for regist me
	var reply RegistPeerNodeReply
	err = client.Call("Raft.RegistPeerNode", myInfo, &reply)
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	// regist5 peer
	service.node.addPeer(peer.Id, &RaftPeerNode{
		id:      peer.Id,
		address: peer.Address,
		client:  client,
	})

	log.WithField("network", "network.ConnectToPeer").Info(goidForlog()+"result : ", reply.Regist)
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
