/*
MIT License

Copyright (c) 2024 ISSuh

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

package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/ISSuh/raft/internal/config"
	"github.com/ISSuh/raft/internal/message"
	"github.com/ISSuh/raft/internal/net"
	"github.com/stretchr/testify/require"
)

var defaultTestConfig config.Config = config.Config{
	Raft: config.RaftConfig{
		Cluster: config.ClussterConfig{
			Address: config.Address{
				Ip:   "127.0.0.1",
				Port: 33157,
			},
		},
		Node: config.NodeConfig{
			Id: 0,
			Address: config.Address{
				Ip:   "127.0.0.1",
				Port: 33158,
			},
			Event: config.Event{
				Timeout: 1000,
			},
			Transport: config.Transport{
				RequestTimeout: 1000,
			},
		},
	},
}

type MockClusterForRpcTransporter struct {
	transporter net.Transporter
}

type MockNodeForRpcTransporter struct {
	transporter net.Transporter
}

func TestServe(t *testing.T) {
	h := &MockRpcHandler{}
	r := NewRpcTransporter(
		defaultTestConfig.Raft.Node.Address, defaultTestConfig.Raft.Node.Transport, h,
	)

	require.NotNil(t, r)

	c, cancel := context.WithCancel(context.Background())
	require.Nil(t, r.Serve(c))

	time.Sleep(1 * time.Second)

	cancel()
}

func TestServeFail(t *testing.T) {
	t.Run("invalid ip", func(t *testing.T) {
		c := defaultTestConfig
		c.Raft.Node.Address.Ip = "999.2.2.1"

		h := &MockRpcHandler{}
		r := NewRpcTransporter(
			c.Raft.Node.Address, c.Raft.Node.Transport, h,
		)

		require.NotNil(t, r)

		err := r.Serve(context.Background())
		require.Error(t, err)
	})

	t.Run("invalid port", func(t *testing.T) {
		c := defaultTestConfig
		c.Raft.Node.Address.Port = 12837129803

		h := &MockRpcHandler{}
		r := NewRpcTransporter(
			c.Raft.Node.Address, c.Raft.Node.Transport, h,
		)

		require.NotNil(t, r)

		err := r.Serve(context.Background())
		require.Error(t, err)
	})
}

func TestStopAndWait(t *testing.T) {
	h := &MockRpcHandler{}
	config := defaultTestConfig
	config.Raft.Node.Address.Port += 10

	r := NewRpcTransporter(
		config.Raft.Node.Address, config.Raft.Node.Transport, h,
	)

	require.NotNil(t, r)
	require.Nil(t, r.Serve(context.Background()))

	time.Sleep(1 * time.Second)

	r.StopAndWait()
}

func TestConnectCluster(t *testing.T) {
	h := &MockRpcHandler{}
	c, cancel := context.WithCancel(context.Background())

	clusterConfig := defaultTestConfig
	clusterConfig.Raft.Cluster.Address.Port += 15
	cluster := NewRpcTransporter(clusterConfig.Raft.Cluster.Address, clusterConfig.Raft.Cluster.Transport, h)
	require.NotNil(t, cluster)
	require.Nil(t, cluster.Serve(c))

	nodeConfig := defaultTestConfig
	nodeConfig.Raft.Node.Address.Port += 16
	r := NewRpcTransporter(nodeConfig.Raft.Node.Address, nodeConfig.Raft.Node.Transport, h)
	require.NotNil(t, r)

	err := r.Serve(c)
	require.NoError(t, err)

	requester, err := r.ConnectCluster(clusterConfig.Raft.Cluster.Address)
	require.NotNil(t, requester)
	require.Nil(t, err)

	cancel()
}

func TestConnectClusterFail(t *testing.T) {
	h := &MockRpcHandler{}
	c, cancel := context.WithCancel(context.Background())

	t.Run("invalid cluster address", func(t *testing.T) {
		config := defaultTestConfig
		config.Raft.Cluster.Address.Ip = "999.0.0.1"
		config.Raft.Node.Address.Port += 20
		r := NewRpcTransporter(config.Raft.Node.Address, config.Raft.Node.Transport, h)
		require.NotNil(t, r)

		err := r.Serve(c)
		require.NoError(t, err)

		requester, err := r.ConnectCluster(config.Raft.Cluster.Address)
		require.Nil(t, requester)
		require.Error(t, err)

		r.StopAndWait()
	})

	cancel()
}

func TestConnectNode(t *testing.T) {
	h := &MockRpcHandler{}
	c, cancel := context.WithCancel(context.Background())

	peerConfig := defaultTestConfig
	peerConfig.Raft.Node.Id += 6
	peerConfig.Raft.Node.Address.Port += 56
	peerNode := NewRpcTransporter(peerConfig.Raft.Node.Address, peerConfig.Raft.Node.Transport, h)
	require.NotNil(t, peerNode)

	err := peerNode.Serve(c)
	require.NoError(t, err)

	config := defaultTestConfig
	config.Raft.Node.Id += 5
	config.Raft.Node.Address.Port += 50
	r := NewRpcTransporter(config.Raft.Node.Address, config.Raft.Node.Transport, h)
	require.NotNil(t, r)

	err = r.Serve(c)
	require.NoError(t, err)

	peerNodeMeta := &message.NodeMetadata{
		Address: &message.Address{
			Ip:   defaultTestConfig.Raft.Node.Address.Ip,
			Port: int32(defaultTestConfig.Raft.Node.Address.Port),
		},
	}

	requester, err := r.ConnectNode(peerNodeMeta)
	require.NotNil(t, requester)
	require.Nil(t, err)

	cancel()
}

func TestConnectNodeFail(t *testing.T) {
	h := &MockRpcHandler{}
	c, cancel := context.WithCancel(context.Background())

	t.Run("invalid node ip", func(t *testing.T) {
		invalidIp := "999.0.0.1"
		config := defaultTestConfig
		config.Raft.Node.Id += 5
		config.Raft.Node.Address.Port += 35

		r := NewRpcTransporter(config.Raft.Node.Address, config.Raft.Node.Transport, h)
		require.NotNil(t, r)
		require.Nil(t, r.Serve(c))

		peerNodeMeta := &message.NodeMetadata{
			Address: &message.Address{
				Ip:   invalidIp,
				Port: int32(defaultTestConfig.Raft.Node.Address.Port),
			},
		}

		requester, err := r.ConnectNode(peerNodeMeta)
		require.Nil(t, requester)
		require.Error(t, err)

		r.StopAndWait()
	})

	t.Run("invalid node port", func(t *testing.T) {
		invalidPort := int32(-11)
		config := defaultTestConfig
		config.Raft.Node.Id += 5
		config.Raft.Node.Address.Port += 40

		r := NewRpcTransporter(config.Raft.Node.Address, config.Raft.Node.Transport, h)
		require.NotNil(t, r)
		require.Nil(t, r.Serve(c))

		peerNodeMeta := &message.NodeMetadata{
			Address: &message.Address{
				Ip:   defaultTestConfig.Raft.Node.Address.Ip,
				Port: invalidPort,
			},
		}

		requester, err := r.ConnectNode(peerNodeMeta)
		require.Nil(t, requester)
		require.Error(t, err)

		r.StopAndWait()
	})

	cancel()
}
