package raft

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func timer(min time.Duration, max time.Duration) <-chan time.Time {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	target := min
	duration := max - min
	if duration > 0 {
		target += time.Duration(rand.Int63n(int64(duration)))
	}
	return time.After(target)
}

func goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}

	return id
}

func goidForlog() string {
	return "[" + strconv.Itoa(goid()) + "] "
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
