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
	mutex      sync.Mutex
}

func NewLogs() *Logs {
	return &Logs{
		entries:     make([]*message.LogEntry, 0),
		commitIndex: -1,
		nextIndex:   map[int32]int64{},
		matchIndex:  map[int32]int64{},
	}
}

func (l *Logs) Len() int {
	return len(l.entries)
}

func (l *Logs) AppendLog(entires []*message.LogEntry) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.entries = append(l.entries, entires...)
}

func (l *Logs) AppendLogSinceToIndex(index int64, entires []*message.LogEntry) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if index != 0 && l.Len() <= int(index) {
		return fmt.Errorf("[AppendLogSinceToIndex] logIndex bigger than len of entries. logIndex : %d, len : %d", index, l.Len())
	}

	l.entries = append(l.entries[:index], entires...)
	return nil
}

func (l *Logs) NextIndex(peerId int32) int64 {
	return l.nextIndex[peerId]
}

func (l *Logs) UpdateNextIndex(peerId int32, index int64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.nextIndex[peerId] = index
}

func (l *Logs) MatchIndex(peerId int32) int64 {
	return l.matchIndex[peerId]
}

func (l *Logs) UpdateMatchIndex(peerId int32, index int64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.matchIndex[peerId] = index
}

func (l *Logs) EntryTerm(logIndex int64) (uint64, error) {
	if l.Len() <= int(logIndex) {
		return 0, fmt.Errorf("[Term] logIndex bigger than len of entries. logIndex : %d, len : %d", logIndex, l.Len())
	}
	return l.entries[logIndex].Term, nil
}

func (l *Logs) CommitIndex() int64 {
	return l.commitIndex
}

func (l *Logs) UpdateCommitIndex(index int64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.commitIndex = index
}

func (l *Logs) Truncate(begin int64) error {
	if begin < 0 {
		return fmt.Errorf("[Truncate] invalid index. begin : %d", begin)
	}
	l.entries = l.entries[begin:]
	return nil
}

func (l *Logs) Since(begin int64) ([]*message.LogEntry, error) {
	end := int64(l.Len() - 1)
	return l.Range(begin, end)
}

func (l *Logs) Range(begin, end int64) ([]*message.LogEntry, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if begin < 0 || end >= int64(l.Len()) {
		return nil, fmt.Errorf("[Range] invalid index. begin : %d, end : %d, len : %d", begin, end, l.Len())
	}
	return l.entries[begin:end], nil
}
