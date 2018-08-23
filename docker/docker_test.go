package docker

import (
	"testing"
)

var name = "library/ubuntu"
var _, _, token, _ = RegistryAuthenticate(name, "*")

func TestAuthentication(t *testing.T) {
	t.Skip("skip test")
	ret, code, token, err := RegistryAuthenticate(name, "pull")
	t.Log(ret, code, token, err)
}

func TestGetCatalog(t *testing.T) {
	t.Skip("skip test")
	t.Log(GetCatalog(token))
}

func TestPullManifest(t *testing.T) {
	//t.Skip("skip test")
	t.Log(PullManifest(name, "latest", token))
}

func TestDockerV2(t *testing.T) {
	t.Skip("skip test")
	t.Log(V2Available())
}

func TestListTags(t *testing.T) {
	t.Skip("skip test")
	t.Log(ListTags(name, token))
}
