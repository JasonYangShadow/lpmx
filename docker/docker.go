package docker

import (
	"fmt"
	"github.com/heroku/docker-registry-client/registry"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DOCKER_URL  = "https://registry-1.docker.io"
	SETTING_URL = "https://raw.githubusercontent.com/JasonYangShadow/LPMXSettingRepository/master/"
)

func ListRepositories(username string, pass string) ([]string, *Error) {
	log.SetOutput(ioutil.Discard)
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return nil, cerr
	}
	repo, err := hub.Repositories()
	if err != nil {
		cerr := ErrNew(err, "query docker repositories failure")
		return nil, cerr
	}
	return repo, nil
}

func ListTags(username string, pass string, name string) ([]string, *Error) {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") {
		name = "library/" + name
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return nil, cerr
	}
	tags, err := hub.Tags(name)
	if err != nil {
		cerr := ErrNew(err, "query docker tags failure")
		return nil, cerr
	}
	return tags, nil
}

func GetDigest(username string, pass string, name string, tag string) (string, *Error) {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") {
		name = "library/" + name
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return "", cerr
	}
	digest, err := hub.ManifestDigest(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker digest failure")
		return "", cerr
	}
	return digest.String(), nil
}

func DeleteManifest(username string, pass string, name string, tag string) *Error {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") {
		name = "library/" + name
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return cerr
	}
	digest, err := hub.ManifestDigest(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker digest failure")
		return cerr
	}
	err = hub.DeleteManifest(name, digest)
	if err != nil {
		cerr := ErrNew(err, "delete docker manifest failure")
		return cerr
	}
	return nil
}

func DownloadLayers(username string, pass string, name string, tag string, folder string) (map[string]int64, *Error) {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") {
		name = "library/" + name
	}
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return nil, err
		}
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return nil, cerr
	}
	man, err := hub.ManifestV2(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker manifest failure")
		return nil, cerr
	}
	data := make(map[string]int64)
	for _, element := range man.Layers {
		dig := element.Digest
		reader, err := hub.DownloadLayer(name, dig)
		if err != nil {
			cerr := ErrNew(err, "download docker layers failure")
			return nil, cerr
		}
		defer reader.Close()
		if strings.HasSuffix(folder, "/") {
			folder = strings.TrimSuffix(folder, "/")
		}
		filename := folder + "/" + strings.TrimPrefix(dig.String(), "sha256:")
		to, err := os.Create(filename)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("create file %s failure", filename))
			return nil, cerr
		}
		defer to.Close()
		fmt.Println(fmt.Sprintf("Downloading file with type: %s, size: %d, destination: %s", element.MediaType, element.Size, filename))

		//printing download percentage using anonymous functions
		go func(filename string, size int64) {
			f, err := os.Open(filename)
			if err != nil {
				return
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil {
				return
			}
			curr_size := fi.Size()
			for curr_size < size {
				percentage := int(float64(curr_size) / float64(size) * 100)
				fmt.Printf("Downloading... %d/%d [%d/100 complete]", curr_size, size, percentage)
				time.Sleep(time.Second)
				fi, err = f.Stat()
				curr_size = fi.Size()
				fmt.Printf("\r")
			}
		}(filename, element.Size)

		if _, err := io.Copy(to, reader); err != nil {
			cerr := ErrNew(err, fmt.Sprintf("copy file %s content failure", filename))
			return nil, cerr
		}
		data[filename] = element.Size
	}
	return data, nil
}

func DownloadSetting(name string, tag string, folder string) *Error {
	filepath := fmt.Sprintf("%s/setting.yml", folder)
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return err
		}
	}
	out, err := os.Create(filepath)
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("%s file create error", filepath))
		return cerr
	}
	defer out.Close()

	http_req := fmt.Sprintf("%s/%s/%s/setting.yml", SETTING_URL, name, tag)
	resp, err := http.Get(http_req)
	if err != nil {
		cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
		return cerr
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", http_req, filepath))
		return cerr
	}
	return nil
}
