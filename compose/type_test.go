package compose

import (
	error2 "github.com/JasonYangShadow/lpmx/error"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestLoadSimpleYamlSuccess(t *testing.T) {
	yamlFile, err := ioutil.ReadFile("simple_success.yaml")
	if err != nil {
		t.Error(err)
		return
	}

	var topLevel TopLevel
	err = yaml.Unmarshal(yamlFile, &topLevel)
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, topLevel.Version, "1", "should be equal to 1")
	assert.Equal(t, len(topLevel.Apps), 2, "should have 2 elements")
}

func TestLoadSimpleYamlFailure(t *testing.T) {
	yamlFile, err := ioutil.ReadFile("simple_failure.yaml")
	if err != nil {
		t.Error(err)
		return
	}

	var topLevel TopLevel
	err = yaml.Unmarshal(yamlFile, &topLevel)
	if err != nil {
		t.Error(err)
		return
	}

	verr := topLevel.validate()
	assert.NotNil(t, verr, "expected error occurs")
	assert.Equal(t, verr.Err, error2.ErrMismatch, "should be mismatch error")
}
