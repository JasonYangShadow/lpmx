package yaml

import (
	"testing"
)

const CONFIG string = "test"

func TestYaml1(t *testing.T) {
	setLocalValue(CONFIG, "environment", "value1")
	real_result, err := getLocalValue(CONFIG, "environment")
	if real_result != "value1" || err != nil {
		t.Fatalf("real: %s => expect: %s", real_result, "value1")
	}
}

func TestYaml2(t *testing.T){
  setValue(CONFIG,[]string{".","~/go"},"environment","value1")
  real_result, err := getValue(CONFIG,[]string{".","~/go"},"environment")
  if real_result != "value1" || err != nil{
    t.Fatalf("real: %s => expect: %s", real_result, "value1")
  }
}
