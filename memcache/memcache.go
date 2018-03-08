package memcache

import (
	"fmt"
	. "github.com/bradfitz/gomemcache/memcache"
	. "github.com/jasonyangshadow/lpmx/error"
	"strings"
)

const (
	MEM_SERVER        = "localhost"
	MEM_PORT          = 11211
	MEMCACHED_MAX_KEY = 256
)

//InitServer is used for initializing memcache server using default configuration
func InitServer() (*Client, *Error) {
	server := fmt.Sprintf("%s:%d", MEM_SERVER, MEM_PORT)
	client := New(server)
	if client == nil {
		err := ErrNew(ErrServerError, fmt.Sprintf("can't create server through the config %s:%d", MEM_SERVER, MEM_PORT))
		return nil, &err
	}
	return client, nil
}

//InitServers is used for initializing memcache servers using parameters
func InitServers(server ...string) (*Client, *Error) {
	client := New(strings.Join(server, ","))
	if client == nil {
		err := ErrNew(ErrServerError, fmt.Sprintf("can't create server through the config %s", server))
		return nil, &err
	}
	return client, nil
}

func GetStrValue(c *Client, key string) (string, *Error) {
	item, err := c.Get(key)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("getStrValue returns error: %s", err.Error()))
		return "", &cerr
	}
	return string(item.Value[:]), nil
}

func SetStrValue(c *Client, key string, value string) *Error {
	item := &Item{Key: key, Value: []byte(value)}
	err := c.Set(item)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("setStrValue returns error: %s", err.Error()))
		return &cerr
	}
	return nil
}

func DeleteByKey(c *Client, key string) *Error {
	err := c.Delete(key)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("deleteByKey returns error: %s", err.Error()))
		return &cerr
	}
	return nil
}
