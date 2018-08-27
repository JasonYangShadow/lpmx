package docker

import (
	"testing"
)

func TestAuthentication(t *testing.T) {
	//t.Skip("skip test")
	err := DownloadLayers("", "", "library/ubuntu", "latest", "/tmp")
	if err != nil {
		t.Error(err)
	}
}
