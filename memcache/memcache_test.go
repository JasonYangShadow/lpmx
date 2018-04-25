package memcache

import (
	"testing"
)

func TestMem1(t *testing.T) {
	mem, err := MInitServer()
	if err == nil {
		mem.MSetStrValue(client, "key", "value")
		value, _ := mem.MGetStrValue(client, "key")
		t.Log(value)
		if value != "value" {
			t.Fatalf("real: %s => expect : %s", value, "value")
		}
	}
}
