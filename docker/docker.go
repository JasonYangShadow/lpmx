package docker

import (
	"fmt"
	"github.com/heroku/docker-registry-client/registry"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/utils"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const (
	DOCKER_URL = "https://registry-1.docker.io"
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

func DownloadLayers(username string, pass string, name string, tag string, folder string) *Error {
	log.SetOutput(ioutil.Discard)
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return err
		}
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return cerr
	}
	man, err := hub.ManifestV2(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker manifest failure")
		return cerr
	}
	for _, element := range man.Layers {
		dig := element.Digest
		reader, err := hub.DownloadLayer(name, dig)
		if err != nil {
			cerr := ErrNew(err, "download docker layers failure")
			return cerr
		}
		defer reader.Close()
		if strings.HasSuffix(folder, "/") {
			folder = strings.TrimSuffix(folder, "/")
		}
		filename := folder + "/" + strings.TrimPrefix(dig.String(), "sha256:")
		to, err := os.Create(filename)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("create file %s failure", filename))
			return cerr
		}
		defer to.Close()
		fmt.Println(fmt.Sprintf("Downloading file with type: %s, size: %d, destination: %s", element.MediaType, element.Size, filename))
		if _, err := io.Copy(to, reader); err != nil {
			cerr := ErrNew(err, fmt.Sprintf("copy file %s content failure", filename))
			return cerr
		}
	}
	return nil
}
