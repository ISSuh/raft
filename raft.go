package raft

type RaftService struct {
	id      int
	node    *RaftNode
	running bool

	network *NetworkService
}

func NewRaftService(id int) *RaftService {
	node := NewRafeNode()
	service := &RaftService{
		id:      id,
		node:    node,
		network: NewNetworkService(node),
	}
	return service
}

func (service *RaftService) Run(port string, peers []PeerInfo) {
	service.running = true
	service.network.Serve(port)

}

func (service *RaftService) Stop() {
	service.running = false
}

func (service *RaftService) IsRunning() bool {
	return service.running
}
