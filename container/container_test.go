package container

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	. "github.com/JasonYangShadow/lpmx/msgpack"
	. "github.com/JasonYangShadow/lpmx/utils"
)

func TestContainerMarshal(t *testing.T) {
	t.Skip("skip test")
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
	t.Skip("skip test")
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

func TestJsonUnmarshal(t *testing.T) {
	//t.Skip("skip test")
	tmpdir := "/tmp"
	uerr := Untar("/tmp/ubuntu.tar", tmpdir, false)
	if uerr != nil {
		t.Error(uerr)
	}

	var dockerSaveInfos []DockerSaveInfo
	manifest_file := fmt.Sprintf("%s/manifest.json", tmpdir)
	b, berr := ioutil.ReadFile(manifest_file)
	if berr != nil {
		t.Error(berr)
	}

	err := json.Unmarshal(b, &dockerSaveInfos)
	if err != nil {
		t.Error(err)
	}

	fmt.Println(dockerSaveInfos)
}
