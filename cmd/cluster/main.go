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
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type NodeInfo struct {
	Id      int    `json:"id"`
	Address string `json:"address"`
}

type NodeManager struct {
	idCounter int
	nodes     map[int]*NodeInfo
}

func NewNodeManager() *NodeManager {
	return &NodeManager{
		idCounter: 0,
		nodes:     make(map[int]*NodeInfo),
	}
}

func (manager *NodeManager) addNewNode(id int, newNode *NodeInfo) bool {
	if _, ok := manager.nodes[id]; ok {
		log.Println("Already exist node. ", id, " / ", newNode.Address)
		return false
	}

	manager.nodes[id] = newNode
	return true
}

func (manager *NodeManager) removeNode(id int) {
	if _, ok := manager.nodes[id]; ok {
		delete(manager.nodes, id)
	}
}

func (manager *NodeManager) node(id int) *NodeInfo {
	if _, ok := manager.nodes[id]; !ok {
		log.Println("node - not exist node. ", id)
		return nil
	}
	return manager.nodes[id]
}

func (manager *NodeManager) nodeList() []NodeInfo {
	var list []NodeInfo
	for _, node := range manager.nodes {
		list = append(list, *node)
	}
	return list
}

type ClusterHandler struct {
	manager *NodeManager
}

func NewClusterHandler() *ClusterHandler {
	return &ClusterHandler{
		manager: NewNodeManager(),
	}
}

func (handler *ClusterHandler) NewNode(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	path := params["id"]
	var id int
	var err error
	if id, err = strconv.Atoi(path); err != nil {
		response(w, "Bad Request. invalid path", http.StatusBadRequest)
		return
	}

	log.Println("NewNode - id : ", id)

	var node NodeInfo
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&node)
	if err != nil {
		log.Println("NewNode - invalid message,. ", r.Body)
		response(w, "Bad Request. invalid message", http.StatusBadRequest)
		return
	}

	if !handler.manager.addNewNode(id, &node) {
		response(w, "Bad Request. already exist", http.StatusBadRequest)
	}
	response(w, "", http.StatusOK)
}

func (handler *ClusterHandler) RemoveNode(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	path := params["id"]
	var id int
	var err error
	if id, err = strconv.Atoi(path); err != nil {
		response(w, "Bad Request. invalid path", http.StatusBadRequest)
		return
	}

	log.Println("RemoveNode - id : ", id)

	handler.manager.removeNode(id)
	response(w, "", http.StatusOK)
}

func (handler *ClusterHandler) Node(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	path := params["id"]
	var id int
	var err error
	if id, err = strconv.Atoi(path); err != nil {
		response(w, "Bad Request. invalid path", http.StatusBadRequest)
		return
	}

	log.Println("Node - id : ", id)
	node := handler.manager.node(id)
	if node == nil {
		response(w, "Bad Request. invalid id", http.StatusBadRequest)
		return
	}

	body, err := json.Marshal(node)
	if err != nil {
		response(w, "Bad Request. data marshaling fail", http.StatusBadRequest)
		return
	}

	response(w, string(body), http.StatusOK)
}

func (handler *ClusterHandler) NodeList(w http.ResponseWriter, r *http.Request) {
	log.Println("NodeList")

	list := handler.manager.nodeList()
	if len(list) <= 0 {
		response(w, "", http.StatusOK)
		return
	}

	body, err := json.Marshal(list)
	if err != nil {
		response(w, "Bad Request. data marshaling fail", http.StatusBadRequest)
		return
	}
	response(w, string(body), http.StatusOK)
}

func response(w http.ResponseWriter, message string, httpStatusCode int) {
	w.WriteHeader(httpStatusCode)
	if http.StatusOK == httpStatusCode {
		w.Header().Set("Content-Type", "application/json")
	}
	w.Write([]byte(message))
}

func main() {
	log.Println("Example Raft Cluster")
	clusterHandler := NewClusterHandler()

	router := mux.NewRouter()
	router.HandleFunc("/node/{id}", clusterHandler.Node).Methods("GET")
	router.HandleFunc("/node/{id}", clusterHandler.NewNode).Methods("POST")
	router.HandleFunc("/node/{id}", clusterHandler.RemoveNode).Methods("DELETE")
	router.HandleFunc("/node", clusterHandler.NodeList).Methods("GET")

	http.Handle("/", router)
	err := http.ListenAndServe(":33660", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
