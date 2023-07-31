package raft

import (
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

type RaftService struct {
	id      int
	node    *RaftNode
	running bool

	network *NetworkService

	testBlock chan bool
}

func NewRaftService(id int) *RaftService {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		FieldsOrder:     []string{"network", "node", "peernode"},
		TimestampFormat: "[2006:01:02 15:04:05.000]",
	})

	node := NewRafeNode(id)
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

	service.node.Run()

	// for test
	<-service.testBlock
}

func (service *RaftService) Stop() {
	service.running = false

}

func (service *RaftService) IsRunning() bool {
	return service.running
}
