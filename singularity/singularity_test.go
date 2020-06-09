package singularity

import (
	"testing"
)

func TestLoadSif(t *testing.T) {
	//t.Skip("skip")
	err := ExtractSquashfs("testdata/ubuntu.sif", "/tmp/file.squashfs")
	if err != nil {
		t.Error(err)
	}
	err = Unsquashfs("/tmp/file.squashfs", "/tmp/folder")
	if err != nil {
		t.Error(err)
	}
}
