package container

import (
	. "github.com/jasonyangshadow/lpmx/msgpack"
	"testing"
)

func TestContainerMarshal(t *testing.T) {
	var con Container
	con.ConfigPath = "/tmp/lpmx_test"
	t.Log(con)
	data, err := StructMarshal(con)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(data)
	}

	var con_new Container
	err = StructUnmarshal(data, &con_new)
	if err != nil {
		t.Error(err)
	} else {
		t.Log(con_new)
	}
}

func TestContainer(t *testing.T) {
	dir := "/tmp/lpmx_test"
	config := "./setting.yml"
	err := Run(dir, config)
	if err != nil {
		t.Error(err)
	}
}
