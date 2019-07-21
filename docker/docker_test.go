package docker

import (
	"io"
	"os"
	"testing"
)

func TestGetToken(t *testing.T) {
	t.Skip("skip test")
	token, err := GetToken("JasonYangShadow/ubuntu", "JasonYangShadow", "", "push,pull")
	if err != nil {
		t.Error(err)
	} else {
		b, err := UploadBlob("JasonYangShadow/ubuntu", token, "/tmp/jRAT9GNac5.tar.gz")
		if err != nil {
			t.Error(err)
		} else {
			t.Logf("********** %v\n", b)
		}
	}
}

func TestHasBlob(t *testing.T) {
	t.Skip("skip test")
	token, _ := GetToken("JasonYangShadow/ubuntu", "JasonYangShadow", "", "push,pull")
	b, berr := HasBlob("JasonYangShadow/ubuntu", token, "45e43933efa9dab764a881ee4a87b4ffde3965584cd03b76f51d17de4b538ee0")
	if berr != nil {
		t.Errorf("**** error %s", berr)
	} else {
		t.Logf("****** %v", b)
	}
}

func TestDownloadBlob(t *testing.T) {
	t.Skip("skip test")
	token, _ := GetToken("JasonYangShadow/ubuntu", "JasonYangShadow", "", "push,pull")
	b, berr := DownloadBlob("JasonYangShadow/ubuntu", token, "45e43933efa9dab764a881ee4a87b4ffde3965584cd03b76f51d17de4b538ee0")
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
	t.Skip("skip test")
	err := UploadManifests("JasonYangShadow", "", "JasonYangShadow/ubuntu", "test", "45e43933efa9dab764a881ee4a87b4ffde3965584cd03b76f51d17de4b538ee0", "ubuntu:16.04")
	if err != nil {
		t.Error(err)
	}
}

func TestDownloadGithub(t *testing.T) {
	//t.Skip("skip test")
	SETTING_URL := "https://raw.githubusercontent.com/JasonYangShadow/LPMXSettingRepository/master"
	yaml := "/tmp/distro.management.yml"
	url, err := GenGithubURLfromYaml("redhat", "6.1", SETTING_URL, yaml, "dependency.tar.gz")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(url)
	}
}

func TestDownloadGithubPlus(t *testing.T) {
	t.Skip("skip test")
	SETTING_URL := "https://raw.githubusercontent.com/JasonYangShadow/LPMXSettingRepository/master"
	yaml := "/tmp/distro.management.yml"
	err := DownloadFilefromGithubPlus("linuxmint", "14.04", "dependency.tar.gz", SETTING_URL, "/tmp", yaml)
	if err != nil {
		t.Error(err)
	}
}
