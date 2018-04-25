package memcache

import (
	"fmt"
	. "github.com/bradfitz/gomemcache/memcache"
	. "github.com/jasonyangshadow/lpmx/error"
	"strings"
)

type MemcacheInst struct {
	MemcacheServerList string
	MemcacheServerPort string
	ClientInst         *Client
}

//InitServer is used for initializing memcache server using default configuration
func MInitServer() (*MemcacheInst, *Error) {
	var mem MemcacheInst
	mem.MemcacheServerList = "locaohost"
	mem.MemcacheServerPort = "11211"
	server := fmt.Sprintf("%s:%s", mem.MemcacheServerList[0], mem.MemcacheServerPort)
	client := New(server)
	if client == nil {
		err := ErrNew(ErrServerError, fmt.Sprintf("can't create server through the config %s:%d", mem.MemcacheServerList, mem.MemcacheServerPort))
		return nil, &err
	}
	mem.ClientInst = client
	return &mem, nil
}

//InitServers is used for initializing memcache servers using parameters
func MInitServers(server ...string) (*MemcacheInst, *Error) {
	var mem MemcacheInst
	mem.MemcacheServerList = strings.Join(server, ",")
	client := New(strings.Join(server, ","))
	if client == nil {
		err := ErrNew(ErrServerError, fmt.Sprintf("can't create server through the config %s", server))
		return nil, &err
	}
	mem.ClientInst = client
	return &mem, nil
}

func (mem *MemcacheInst) MGetStrValue(key string) (string, *Error) {
	item, err := mem.ClientInst.Get(key)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("getStrValue returns error: %s", err.Error()))
		return "", &cerr
	}
	return string(item.Value[:]), nil
}

func (mem *MemcacheInst) MSetStrValue(key string, value string) *Error {
	item := &Item{Key: key, Value: []byte(value)}
	err := mem.ClientInst.Set(item)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("setStrValue returns error: %s", err.Error()))
		return &cerr
	}
	return nil
}

func (mem *MemcacheInst) MDeleteByKey(key string) *Error {
	err := mem.ClientInst.Delete(key)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("deleteByKey returns error: %s", err.Error()))
		return &cerr
	}
	return nil
}
