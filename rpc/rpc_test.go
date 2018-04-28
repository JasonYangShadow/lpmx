package rpc

import (
	"testing"
	"time"
)

func TestRPCServer(t *testing.T) {
	port, err := StartServer()
	if err != nil {
		t.Error(err)
	}
	t.Log(port)
}

func TestRPCClient(t *testing.T) {
	var req Request
	var res Response
	req.Timeout = time.Second * time.Duration(10)
	req.Cmd = "ls"
	req.Args = []string{"-al"}
	req.Dir = "/tmp/lpmx_test"
	ch, err := StartClient(req, &res, 11221)
	if err != nil {
		t.Error(err)
	}
	t.Log(ch)
}
