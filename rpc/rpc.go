package rpc

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/paeudo"
	. "github.com/jasonyangshadow/lpmx/utils"
	"net"
	"net/http"
	"net/rpc"
	"time"
)

const (
	MIN = 10000
	MAX = 15000
)

type Request struct {
	UId     string
	Timeout time.Duration
	Cmd     string
	Env     map[string]string
	Dir     string
	Args    []string
}

type Response struct {
	UId string
	Pid int
}

type RPC struct{}

func (server *RPC) Exec(req Request, res *Response) error {
	cmd, err := ProcessContextEnv(req.Cmd, req.Env, req.Dir, req.Timeout, req.Args[0:]...)
	if err != nil {
		return err.Err
	}
	res.UId = req.UId
	res.Pid = cmd.Process.Pid
}

func StartServer() (int, error) {
	r := new(RPC)
	server := rpc.NewServer()
	server.RegisterName("LPMXRPC", r)
	server.HandleHTTP("/", nil)
	con, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return -1, err
	}
	go http.Serve(con, nil)
	return port, nil
}

func StartClient(req Request, res *Response, port int) (*Call, error) {
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	call := client.Go("LPMXRPC.Exec", req, res, nil)
	return call, nil
}
