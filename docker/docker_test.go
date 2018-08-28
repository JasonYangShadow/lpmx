package docker

import (
	"testing"
)

func TestDownloadLayers(t *testing.T) {
	//t.Skip("skip test")
	data, err := DownloadLayers("", "", "library/alpine", "latest", "/tmp")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(data)
	}
}
