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

package log

import (
	"fmt"
	"sync"

	"github.com/ISSuh/raft/internal/message"
)

type Logs struct {
	entries     []*message.LogEntry
	commitIndex int64
	// map[peerId]logIndex
	nextIndex map[int32]int64
	// map[peerId]logIndex
	matchIndex map[int32]int64
	logMutex   sync.Mutex
}

func NewLogs() Logs {
	return Logs{
		entries:     make([]*message.LogEntry, 0),
		commitIndex: -1,
		nextIndex:   map[int32]int64{},
		matchIndex:  map[int32]int64{},
	}
}

func (l *Logs) Len() int {
	return len(l.entries)
}

func (l *Logs) AppendLog(logIndex int64, entires []*message.LogEntry) error {
	if l.Len() <= int(logIndex) {
		return fmt.Errorf("[Logs.AppendLog] logIndex bigger than len of entries. logIndex : %d, len : %d", logIndex, l.Len())
	}

	l.entries = append(l.entries[:logIndex], entires...)
	return nil
}

func (l *Logs) NextIndex(peerId int32) int64 {
	return l.nextIndex[peerId]
}

func (l *Logs) UpdateNextIndex(peerId int32, index int64) {
	l.logMutex.Lock()
	defer l.logMutex.Unlock()
	l.nextIndex[peerId] = index
}

func (l *Logs) Term(logIndex int32) (uint64, error) {
	if l.Len() <= int(logIndex) {
		return 0, fmt.Errorf("[Logs.Term] logIndex bigger than len of entries. logIndex : %d, len : %d", logIndex, l.Len())
	}
	return l.entries[logIndex].Term, nil
}

func (l *Logs) MatchIndex(peerId int32) int64 {
	return l.matchIndex[peerId]
}
