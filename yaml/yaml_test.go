package yaml

import (
	"testing"
	"time"
)

const CONFIG string = "test"

func TestYaml1(t *testing.T) {
	SetLocalValue(CONFIG, "environment", "value1")
	real_result, err := GetLocalValue(CONFIG, "environment")
	if real_result != "value1" || err != nil {
		t.Fatalf("real: %s => expect: %s", real_result, "value1")
	}
}

func TestYaml2(t *testing.T) {
	SetValue(CONFIG, []string{".", "~/go"}, "environment", "value1")
	real_result, err := GetValue(CONFIG, []string{".", "~/go"}, "environment")
	if real_result != "value1" || err != nil {
		t.Fatalf("real: %s => expect: %s", real_result, "value1")
	}
	t.Log(GetMap(CONFIG, []string{".", "~/go"}))
}
