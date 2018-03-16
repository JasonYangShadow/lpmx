package memcache

import (
	"testing"
)

func TestMem1(t *testing.T) {
	client, err := InitServer()
	if err != nil {
		SetStrValue(client, "key", "value")
		value, _ := GetStrValue(client, "key")
		t.Log(value)
		if value != "value" {
			t.Fatalf("real: %s => expect : %s", value, "value")
		}
	}
}
