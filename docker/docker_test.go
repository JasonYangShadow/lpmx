package docker

import (
	"testing"
)

var name = "jasonyangshadow/python"
var _, _, token, _ = RegistryAuthenticate(name, "pull")

func TestPullManifest(t *testing.T) {
	t.Skip("skip test")
	t.Log(PullManifest(name, "latest", token))
}

func TestAuthenticationBasic(t *testing.T) {
	//t.Skip("skip test")
}

func TestDockerV2(t *testing.T) {
	t.Skip("skip test")
	t.Log(V2Available())
}

func TestListTags(t *testing.T) {
	t.Skip("skip test")
	t.Log(ListTags(name, token))
}
