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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	TestId = 0
	TestAddress = "0.0.0.0:33660"
)

func TestNewRaftNode(t *testing.T) {
	node := NewRaftNode(NodeInfo{Id: TestId, Address: TestAddress})
	assert.NotEqual(t, node, (*RaftNode)(nil))
	assert.Equal(t, node.state, FOLLOWER)
	assert.Equal(t, node.currentTerm(), uint64(0))
}

func TestRunAndStop(t *testing.T) {
	node := NewRaftNode(NodeInfo{Id: TestId, Address: TestAddress})
	assert.NotEqual(t, node, (*RaftNode)(nil))

	node.Run()
	time.Sleep(100 * time.Millisecond)

	node.Stop()
	assert.Equal(t, node.currentState(), STOP)
}

func TestFollwerWork(t *testing.T) {
	node := NewRaftNode(NodeInfo{Id: TestId, Address: TestAddress})
	assert.NotEqual(t, node, (*RaftNode)(nil))

	node.follwerWork()
	assert.Equal(t, node.currentState(), CANDIDATE)
}