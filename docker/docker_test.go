package docker

import (
	"testing"
)

func TestUploadLayers(t *testing.T) {
	t.Skip("skip test")
	sha, err := UploadLayers("", "", "jasonyangshadow/ubuntu", "nano", "/tmp/jRAT9GNac5.tar.gz")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(sha)
	}
}

func TestUploadManifest(t *testing.T) {
	t.Skip("skip test")
	err := UploadManifests("", "", "jasonyangshadow/ubuntu", "nano")
	if err != nil {
		t.Error(err)
	}
}
