package msgpack

import (
	"fmt"
	"testing"
)

type Test struct {
	id   int
	name string
}

func TestMarshandUnmarsh(t *testing.T) {
	//t.Skip("skip test")
	test := Test{1, "hi"}
	data, err := StructMarshal(test)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)

	var ret Test
	err = StructUnmarshal(data, &ret)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(ret)
}
