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

package main

import (
	"log"
	"net"
	"net/rpc"
	"os"

	"github.com/ISSuh/raft/message"
)

const (
	Localhost = "localhost"
	RpcMethod = "Raft.ApplyEntry"
)

func main() {
	log.Println("RAFT ")
	args := os.Args[1:]
	if len(args) < 1 {
		log.Println("invalid number of arguments")
		return
	}

	port := args[0]
	address := Localhost + ":" + port
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		log.Println("net.ResolveTCPAddr err : ", err)
		return
	}

	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		log.Println("rpc.Dial err : ", err)
		return
	}

	applyEntry := message.ApplyEntry{
		Log: []byte("TEST"),
	}

	var reply bool
	err = client.Call(RpcMethod, &applyEntry, &reply)
	if err != nil {
		log.Println("client.Call err : ", err)
	}

	log.Println("respone: ", reply)
}
