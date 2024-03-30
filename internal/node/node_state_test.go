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

package node

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNodeState(t *testing.T) {
	nodeState := NewNodeState()
	assert.Equal(t, nodeState.state, FOLLOWER)
	assert.Equal(t, nodeState.currentTerm(), uint64(0))
}

func TestSetState(t *testing.T) {
	nodeState := NewNodeState()
	assert.Equal(t, nodeState.currentState(), FOLLOWER)
	assert.Equal(t, nodeState.currentTerm(), uint64(0))

	nodeState.setState(CANDIDATE)
	assert.Equal(t, nodeState.currentState(), CANDIDATE)
}

func TestSetTerm(t *testing.T) {
	nodeState := NewNodeState()
	assert.Equal(t, nodeState.currentState(), FOLLOWER)
	assert.Equal(t, nodeState.currentTerm(), uint64(0))

	nodeState.setTerm(100)
	assert.Equal(t, nodeState.currentTerm(), uint64(100))
}

func TestInscreaseTerm(t *testing.T) {
	nodeState := NewNodeState()
	assert.Equal(t, nodeState.currentState(), FOLLOWER)
	assert.Equal(t, nodeState.currentTerm(), uint64(0))

	nodeState.increaseTerm()
	assert.Equal(t, nodeState.currentTerm(), uint64(1))
}

func TestConcurrencyIncrease(t *testing.T) {
	nodeState := NewNodeState()
	assert.Equal(t, nodeState.currentState(), FOLLOWER)
	assert.Equal(t, nodeState.currentTerm(), uint64(0))

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		for i := 0; i < 1000; i++ {
			nodeState.increaseTerm()
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			nodeState.increaseTerm()
		}
		wg.Done()
	}()

	wg.Wait()
	assert.Equal(t, nodeState.currentTerm(), uint64(2000))
}
