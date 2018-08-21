package docker

import (
	"testing"
)

func TestDockerTokenBasic(t *testing.T) {
	ret, code, token, err := RegistryAuthenticateBasic("jasonyangshadow/python", "pull", "jasonyangshadow", "jason294514")
	t.Log(ret, code, token, err)
}
