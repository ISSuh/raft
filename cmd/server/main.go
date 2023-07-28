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

func clientRequest(method, path string, node *raft.PeerNodeInfo) []raft.PeerNodeInfo {
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

func HttpResponseHandler(resp *http.Response, expectArrayJson bool) []raft.PeerNodeInfo {
	log.Println(resp.StatusCode)

	switch resp.StatusCode {
	case http.StatusOK:
		return parseBody(resp, expectArrayJson)
	default:
		log.Println("http response error")
		return nil
	}
}

func parseBody(resp *http.Response, expectArrayJson bool) []raft.PeerNodeInfo {
	var res []raft.PeerNodeInfo
	if expectArrayJson {
		decoder := json.NewDecoder(resp.Body)
		err := decoder.Decode(&res)
		if err != nil {
			return nil
		}
	} else {
		var item raft.PeerNodeInfo
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

	node := raft.PeerNodeInfo{Id: id, Address: address}
	clientRequest(http.MethodPost, "node/"+idStr, &node)
	service := raft.NewRaftService(id)
	service.Run(address, nodeList)

	service.Stop()
	clientRequest(http.MethodDelete, "node/"+idStr, nil)
}
