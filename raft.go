/*
MIT License

Copyright (c) 2023 ISSuh

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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
