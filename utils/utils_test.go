package utils

import (
	"fmt"
	"testing"
)

func TestGdrive(t *testing.T) {
	url := "https://drive.google.com/file/d/0B_Pq1NjbA3TdQ1dzYzBKYXlNSVU/view?usp=sharing"
	new_url, n_err := GetGDriveDownloadLink(url)
	if n_err != nil {
		t.Error(n_err)
	}
	d_err := DownloadFile(new_url, "/tmp", "file")
	if d_err != nil {
		t.Error(d_err)
	}
}

func TestTar(t *testing.T) {
	t.Skip("test")
	layers := []string{"/tmp"}
	err := TarLayer("/tmp/test", "/tmp", "file", layers)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPid(t *testing.T) {
	t.Skip("skip test")
	b, err := CheckProcessByPid("30742")
	if err != nil {
		t.Error(err)
	} else {
		t.Log(b)
	}
}

func TestUtilsUntar(t *testing.T) {
	t.Skip("skip test")
	err := Untar("/tmp/8e3ba11ec2a2b39ab372c60c16b421536e50e5ce64a0bc81765c2e38381bcff6", "/tmp/alpine")
	if err != nil {
		t.Error(err)
	}
}

func TestUtils1(t *testing.T) {
	t.Skip("skip test")
	file := "/home/jason/.spacemacs"
	ret := FileExist(file)
	fmt.Printf("%t", ret)
}

func TestUtils2(t *testing.T) {
	t.Skip("skip test")
	src := "/tmp/log"
	dst := "/tmp/log.bak"
	val, err := CopyFile(src, dst)
	if val {
		t.Log("successfully copied")
	} else {
		t.Error(err)
	}
}
