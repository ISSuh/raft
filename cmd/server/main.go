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
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/ISSuh/raft"
)

const (
	ClusterApiUrl = "http://localhost:33660/"
	Localhost     = "127.0.0.1"
)

func clientRequest(method, path string, node *raft.NodeInfo) []raft.NodeInfo {
	url := ClusterApiUrl + path
	log.Println("clientRequest - url : ", url)
	json, _ := json.Marshal(node)
	buffer := bytes.NewBuffer(json)

	expectArrayJson := (path == "node")

	client := http.Client{}
	req, err := http.NewRequest(method, url, buffer)
	if err != nil {
		return nil
	}
	res, err := client.Do(req)
	if err != nil {
		return nil
	}
	return HttpResponseHandler(res, expectArrayJson)
}

func HttpResponseHandler(resp *http.Response, expectArrayJson bool) []raft.NodeInfo {
	log.Println(resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusOK:
		return parseBody(resp, expectArrayJson)
	default:
		log.Println("http response error")
		return nil
	}
}

func parseBody(resp *http.Response, expectArrayJson bool) []raft.NodeInfo {
	var res []raft.NodeInfo
	if expectArrayJson {
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&res)
		if err != nil {
			return nil
		}
	} else {
		var item raft.NodeInfo
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&item)
		if err != nil {
			return nil
		}
		res = append(res, item)
	}
	return res
}

func main() {
	log.Println("RAFT ")
	args := os.Args[1:]
	if len(args) < 2 {
		log.Println("invalid number of arguments")
		return
	}

	idStr := args[0]
	var id int
	var err error
	if id, err = strconv.Atoi(idStr); err != nil {
		log.Println("invalid id argument")
		return
	}

	port := args[1]
	address := Localhost + ":" + port

	nodeList := clientRequest(http.MethodGet, "node", nil)
	log.Println("peers : ", nodeList)

	node := raft.NodeInfo{Id: id, Address: address}
	clientRequest(http.MethodPost, "node/"+idStr, &node)

	service := raft.NewRaftService(id, address)
	service.RegistTrasnporter(raft.NewRpcTransporter())

	service.Run()

	peerMap := make(map[int]string)
	for _, node := range nodeList {
		peerMap[node.Id] = node.Address
	}

	service.ConnectToPeers(peerMap)

	service.Stop()
	clientRequest(http.MethodDelete, "node/"+idStr, nil)
}
