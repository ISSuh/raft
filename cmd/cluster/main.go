package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type NodeInfo struct {
	Id      string `json:"id"`
	Address string `json:"address"`
}

type NodeManager struct {
	idCounter int
	nodes     map[string]*NodeInfo
}

func NewNodeManager() *NodeManager {
	return &NodeManager{
		idCounter: 0,
		nodes:     make(map[string]*NodeInfo),
	}
}

func (manager *NodeManager) addNewNode(id string, newNode *NodeInfo) bool {
	if _, ok := manager.nodes[id]; ok {
		log.Println("Already exist node. ", id, " / ", newNode.Address)
		return false
	}

	manager.nodes[id] = newNode
	return true
}

func (manager *NodeManager) removeNode(id string) {
	if _, ok := manager.nodes[id]; ok {
		delete(manager.nodes, id)
	}
}

func (manager *NodeManager) node(id string) *NodeInfo {
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
	id := params["id"]

	log.Println("NewNode - id : ", id)

	var node NodeInfo
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&node)
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
	id := params["id"]

	log.Println("RemoveNode - id : ", id)

	handler.manager.removeNode(id)
	response(w, "", http.StatusOK)
}

func (handler *ClusterHandler) Node(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

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
