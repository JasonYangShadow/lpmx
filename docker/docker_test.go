package docker

import (
	"io"
	"os"
	"testing"
)

func TestGetToken(t *testing.T) {
	t.Skip("skip test")
	token, err := GetToken("jasonyangshadow/ubuntu", "jasonyangshadow", "", "push,pull")
	if err != nil {
		t.Error(err)
	} else {
		b, err := UploadBlob("jasonyangshadow/ubuntu", token, "/tmp/jRAT9GNac5.tar.gz")
		if err != nil {
			t.Error(err)
		} else {
			t.Logf("********** %v\n", b)
		}
	}
}

func TestHasBlob(t *testing.T) {
	t.Skip("skip test")
	token, _ := GetToken("jasonyangshadow/ubuntu", "jasonyangshadow", "", "push,pull")
	b, berr := HasBlob("jasonyangshadow/ubuntu", token, "45e43933efa9dab764a881ee4a87b4ffde3965584cd03b76f51d17de4b538ee0")
	if berr != nil {
		t.Errorf("**** error %s", berr)
	} else {
		t.Logf("****** %v", b)
	}
}

func TestDownloadBlob(t *testing.T) {
	t.Skip("skip test")
	token, _ := GetToken("jasonyangshadow/ubuntu", "jasonyangshadow", "", "push,pull")
	b, berr := DownloadBlob("jasonyangshadow/ubuntu", token, "45e43933efa9dab764a881ee4a87b4ffde3965584cd03b76f51d17de4b538ee0")
	if berr != nil {
		t.Errorf("**** error %s", berr)
	} else {
		out, err := os.Create("/tmp/file")
		if err != nil {
			t.Error(err)
		}
		defer out.Close()
		_, err = io.Copy(out, b)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestUploadManifest(t *testing.T) {
	//t.Skip("skip test")
	err := UploadManifests("jasonyangshadow", "", "jasonyangshadow/ubuntu", "test", "45e43933efa9dab764a881ee4a87b4ffde3965584cd03b76f51d17de4b538ee0", "ubuntu:16.04")
	if err != nil {
		t.Error(err)
	}
}
