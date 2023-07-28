package raft

type RaftService struct {
	id      int
	node    *RaftNode
	running bool

	network *NetworkService

	testBlock chan bool
}

func NewRaftService(id int) *RaftService {
	node := NewRafeNode()
	service := &RaftService{
		id:        id,
		node:      node,
		network:   NewNetworkService(id, node),
		testBlock: make(chan bool),
	}
	return service
}

func (service *RaftService) Run(address string, peers []PeerNodeInfo) {
	service.running = true
	service.network.Serve(address, peers)

	for _, peer := range peers {
		service.network.ConnectToPeer(peer)
	}

	<-service.testBlock
}

func (service *RaftService) Stop() {
	service.running = false

}

func (service *RaftService) IsRunning() bool {
	return service.running
}
