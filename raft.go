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
	log "github.com/sirupsen/logrus"
)

type RaftService struct {
	node    *RaftNode
	transporter Transporter
	testBlock chan bool
}

func NewRaftService(id int, address string) *RaftService {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		FieldsOrder:     []string{"network", "node", "peernode"},
		TimestampFormat: "[2006:01:02 15:04:05.000]",
	})

	node := NewRafeNode(NodeInfo{Id: id, Address: address})
	service := &RaftService{
		node:      node,
		transporter: nil,
		testBlock: make(chan bool),
	}
	return service
}

func (service *RaftService) Node() *RaftNode {
	return service.node
}

func (service *RaftService) RegistTrasnporter(transporter Transporter) {
	service.transporter = transporter;
}

func (service *RaftService) Run(peers []NodeInfo) {
	err := service.transporter.Serve(service.node.info.Address);
	if err != nil {
		return
	}

	myInfo := NodeInfo{
		Id:      service.node.info.Id,
		Address: service.node.info.Address,
	}

	for _, peer := range peers {
		peerNode, err := service.transporter.ConnectToPeer(peer)
		if err != nil {
			continue
		}

		service.node.addPeer(peerNode.id, peerNode)

		var reply RegistPeerNodeReply
		err = peerNode.RegistPeerNode(&myInfo, &reply)
		if err != nil {
			log.WithField("network", "service.Run").Error(goidForlog()+"err : ", err)
		}
	}

	service.node.Run()

	// for test
	<-service.testBlock
}

func (service *RaftService) Stop() {
	service.transporter.Stop()
}

