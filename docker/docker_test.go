package docker

import (
	"fmt"
	"github.com/heroku/docker-registry-client/registry"
	"testing"
)

func TestDownloadLayers(t *testing.T) {
	//t.Skip("skip test")
	url := "https://registry-1.docker.io/"
	username := "" // anonymous
	password := "" // anonymous
	hub, _ := registry.New(url, username, password)
	manifest, err := hub.ManifestV2("/ubuntu", "latest")
	if err != nil {
		t.Error(err)
	}
	fmt.Print(manifest)
}
