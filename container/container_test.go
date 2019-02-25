package container

import (
	. "github.com/JasonYangShadow/lpmx/msgpack"
	. "github.com/JasonYangShadow/lpmx/utils"
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

func TestUnmarshal(t *testing.T) {
	var con Container
	data, err := ReadFromFile("/tmp/lpmx_test/.lpmx/.info")
	if err != nil {
		t.Error(err)
	}
	err = StructUnmarshal(data, &con)
	if err != nil {
		t.Error(err)
	}
	t.Log(con)
}
