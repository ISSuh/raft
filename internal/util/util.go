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

package util

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func Timout(min time.Duration, max time.Duration) time.Duration {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	target := min
	duration := max - min
	if duration > 0 {
		target += time.Duration(rand.Int63n(int64(duration)))
	}
	return target
}

func Timer(min time.Duration, max time.Duration) <-chan time.Time {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	target := min
	duration := max - min
	if duration > 0 {
		target += time.Duration(rand.Int63n(int64(duration)))
	}
	return time.After(target)
}

func Ticker(min time.Duration, max time.Duration) *time.Ticker {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	target := min
	duration := max - min
	if duration > 0 {
		target += time.Duration(rand.Int63n(int64(duration)))
	}
	return time.NewTicker(target)
}

func GoId() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}

	return id
}

func GoidForlog() string {
	return "[" + strconv.Itoa(GoId()) + "] "
}

func Min(a, b int64) int64 {
	if a >= b {
		return b
	}
	return a
}

func Max(a, b int64) int64 {
	if a >= b {
		return a
	}
	return b
}

func BooleanToByteSlice(value bool) []byte {
	if value {
		return []byte{1}
	}
	return []byte{0}
}

func BooleanByteSliceToBool(value []byte) bool {
	if len(value) > 0 && value[0] > 0 {
		return true
	}
	return false
}

func RandRange(min, max int) int {
	return rand.Intn(max-min) + min
}
