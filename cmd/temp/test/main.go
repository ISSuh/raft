package main

import (
	"context"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
)

var address string = "127.0.0.1:33111"

type RpcArgument struct {
	Id      string
	Message interface{}
}

type RpcResponse struct {
	Id      string
	Message interface{}
	Err     error
}

type server struct {
	rpcServer *rpc.Server
	listener  net.Listener
	quit      chan bool
	wg        sync.WaitGroup
}

func (t *server) serveRpcServer(context context.Context) error {
	var err error
	t.listener, err = net.Listen("tcp", address)
	if err != nil {
		return err
	}

	t.runServer(context)
	return nil
}

func (t *server) runServer(context context.Context) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		for {
			conn, err := t.listener.Accept()
			if err != nil {
				select {
				case <-context.Done():
					fmt.Printf("contex cancel\n")
					return
				case <-t.quit:
					fmt.Printf("quit\n")
					return
				default:
					continue
				}
			}

			t.wg.Add(1)
			go func() {
				t.rpcServer.ServeConn(conn)
				t.wg.Done()
			}()
		}
	}()
}

func (t *server) Func(arg *RpcArgument, reply *RpcResponse) error {
	fmt.Printf("[Func] arg : %+v\n", arg)

	reply.Id = arg.Id
	reply.Message = 67
	return nil
}

type client struct {
}

func (t *client) Connect(address string) (*rpc.Client, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	client, err := rpc.Dial(addr.Network(), addr.String())
	if err != nil {
		return nil, err
	}
	return client, nil
}

func main() {
	s := &server{
		rpcServer: rpc.NewServer(),
		quit:      make(chan bool),
	}

	err := s.rpcServer.RegisterName("test", s)
	if err != nil {
		fmt.Printf("err = %s\n", err.Error())
		return
	}

	c, candel := context.WithCancel(context.Background())
	s.serveRpcServer(c)

	time.Sleep(1 * time.Second)

	a := client{}
	cli, err := a.Connect(address)
	if err != nil {
		fmt.Printf("err = %s\n", err.Error())
		return
	}

	fmt.Printf("start client \n")

	value := "test"
	arg := RpcArgument{
		Id:      "0",
		Message: value,
	}
	reply := RpcResponse{}
	err = cli.Call("test.Func", &arg, &reply)
	if err != nil {
		fmt.Printf("err = %s\n", err.Error())
		return
	}

	fmt.Printf("reply = %v\n", reply)

	time.Sleep(1 * time.Second)
	candel()
}
