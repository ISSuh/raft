package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	ClusterApiUrl = "http://localhost:33660/"
	Localhost     = "127.0.0.1"
)

type NodeInfo struct {
	Id      string `json:"id"`
	Address string `json:"address"`
}

func clientRequest(method, path string, node *NodeInfo) (map[string]interface{}, error) {
	url := ClusterApiUrl + path
	log.Println("clientRequest - url : ", url)
	json, _ := json.Marshal(node)
	buffer := bytes.NewBuffer(json)

	client := http.Client{}
	req, err := http.NewRequest(method, url, buffer)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return HttpResponseHandler(res)
}

func HttpResponseHandler(resp *http.Response) (map[string]interface{}, error) {
	switch resp.StatusCode {
	case http.StatusOK:
		return parseBody(resp)
	default:
		err := fmt.Errorf("http response error")
		return nil, err
	}
}

func parseBody(response *http.Response) (map[string]interface{}, error) {
	res := make(map[string]interface{})
	if err := json.NewDecoder(response.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res, nil
}

func main() {
	log.Println("RAFT ")
	args := os.Args[1:]
	id := args[0]
	port := args[1]
	address := Localhost + ":" + port

	test, _ := clientRequest(http.MethodGet, "node", nil)
	log.Println(test)

	node := NodeInfo{Id: id, Address: address}
	test, _ = clientRequest(http.MethodPost, "node/"+id, &node)
	log.Println(test)

	test, _ = clientRequest(http.MethodGet, "node", nil)
	log.Println(test)
}
