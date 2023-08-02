package raft

import (
	"net"
	"net/rpc"
	"sync"

	log "github.com/sirupsen/logrus"
)

type OnRequestVote = func(int) int

type NetworkHandler interface {
	OnRequestVote()
	OnAppendEntries()
}

type Network interface {
	RegistHandler(handler *NetworkHandler)

	RegistPeerNode(arg *PeerNodeInfo, resp *RegistPeerNodeReply)
	RequestVote(arg *RequestVoteArgs, resp *RequestVoteReply)
	AppendEntries(arg *AppendEntriesArgs, resp *AppendEntriesReply)
}

type LogEntry struct {
	Term    uint64
	Command []byte
}

type ApplyEntry struct {
	Command []byte
}

type ApplyEntryReply struct {
	Success bool
}

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

	PrevLogIndex      int64
	PrevLogTerm       uint64
	Entries           []LogEntry
	LeaderCommitIndex int64
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
	proxy.node.onRequestVote(args, reply)
	return nil
}

func (proxy *RPCProxy) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	log.WithField("network", "network.AppendEntries").Info(goidForlog())
	proxy.node.onAppendEntries(args, reply)
	return nil
}

func (proxy *RPCProxy) ApplyEntry(args ApplyEntry, reply *ApplyEntryReply) error {
	log.WithField("network", "network.ApplyEntry").Info(goidForlog())
	reply.Success = proxy.node.ApplyEntry(args.Command)
	return nil
}

type NetworkService struct {
	id      int
	address string

	node      *RaftNode
	rpcServer *rpc.Server
	listener  net.Listener
	rpcProxy  *RPCProxy

	mutex sync.Mutex
	wg    sync.WaitGroup

	quit chan interface{}
}

func NewNetworkService(id int, node *RaftNode) *NetworkService {
	return &NetworkService{
		id:   id,
		node: node,
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

func (service *NetworkService) ConnectToPeer(peerInfo PeerNodeInfo) error {
	log.WithField("network", "network.ConnectToPeer").Info(goidForlog()+"peer : ", peerInfo)
	peerId := peerInfo.Id
	addr, err := net.ResolveTCPAddr("tcp", peerInfo.Address)
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	if peer := service.node.getPeer(peerId); peer != nil {
		log.WithField("network", "network.ConnectToPeer").Warn(goidForlog()+"alread registed. ", peerId)
		return nil
	}

	// connect rpc server
	service.mutex.Lock()
	defer service.mutex.Unlock()
	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	// regist5 peer
	peerNode := &RaftPeerNode{
		id:      peerInfo.Id,
		address: peerInfo.Address,
		client:  client,
	}

	service.node.addPeer(peerInfo.Id, peerNode)

	myInfo := PeerNodeInfo{
		Id:      service.id,
		Address: service.address,
	}

	// notify peer for regist me
	var reply RegistPeerNodeReply
	err = peerNode.RegistPeerNode(myInfo, &reply)
	if err != nil {
		log.WithField("network", "network.ConnectToPeer").Error(goidForlog()+"err : ", err)
		return err
	}

	log.WithField("network", "network.ConnectToPeer").Info(goidForlog()+"result : ", reply.Regist)
	return nil
}
