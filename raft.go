package raft

type RaftService struct {
	node     *RaftNode
	peerNode map[int]*RaftPeerNode
	running  bool

	network *NetworkService
}

func NewRaftService() *RaftService {
	service := &RaftService{
		node:     NewRafeNode(),
		peerNode: make(map[int]*RaftPeerNode, 0),
	}
}

func (service *RaftService) Serve() {
	service.running = true
}

func (service *RaftService) Stop() {
	service.running = false
}

func (service *RaftService) IsRunning() bool {
	return service.running
}
