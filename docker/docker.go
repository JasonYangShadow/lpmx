package docker

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	. "github.com/JasonYangShadow/lpmx/error"
	registry "github.com/JasonYangShadow/lpmx/registry"
	. "github.com/JasonYangShadow/lpmx/utils"
	. "github.com/JasonYangShadow/lpmx/yaml"
	. "github.com/docker/distribution/manifest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/libtrust"
	digest "github.com/opencontainers/go-digest"
)

const (
	DOCKER_URL  = "https://registry-1.docker.io"
	SETTING_URL = "https://raw.githubusercontent.com/JasonYangShadow/LPMXSettingRepository/master"
)

//Docker load image structure
type DockerSaveInfo struct {
	Config   string   //config json file name
	RepoTags []string //tag list
	Layers   []string //layers included inside this image, from lower to higher layers
}

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
	if !strings.Contains(name, "library/") && !strings.Contains(name, "/") {
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
	if !strings.Contains(name, "library/") && !strings.Contains(name, "/") {
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

func MakeManifestV1(registry *registry.Registry, name, tag, layer_sha, base_image string) (*schema1.SignedManifest, *Error) {
	log.SetOutput(ioutil.Discard)

	if !strings.Contains(base_image, "library/") && !strings.Contains(base_image, "/") {
		base_image = "library/" + base_image
	}
	base_info := strings.Split(base_image, ":")
	man_base, man_err := registry.Manifest(base_info[0], base_info[1])
	if man_err != nil {
		cerr := ErrNew(man_err, fmt.Sprintf("unable to parse base image manifest: %s", base_image))
		return nil, cerr
	}

	manifest := schema1.Manifest{
		Versioned: Versioned{
			SchemaVersion: 1,
		},
		Name: name,
		Tag:  tag,
	}
	//modification starts
	//FSLayers assemble
	dig := digest.NewDigestFromHex("sha256", layer_sha)
	new_fslayer := schema1.FSLayer{
		BlobSum: dig,
	}
	layers := []schema1.FSLayer{new_fslayer}
	layers = append(layers, man_base.FSLayers...)
	manifest.FSLayers = layers

	//history assemble
	//replacing the original top V1
	orig_v1 := make(map[string]*json.RawMessage)
	err := json.Unmarshal([]byte(man_base.History[0].V1Compatibility), &orig_v1)
	if err != nil {
		cerr := ErrNew(err, "could not unmarshal V1Compatibility object")
		return nil, cerr
	}

	type v1comp struct {
		ID              string `json:"id"`
		Parent          string `json:"parent,omitempty"`
		Comment         string `json:"comment,omitempty"`
		Created         string `json:"created"`
		ContainerConfig struct {
			Cmd []string
		} `json:"container_config,omitempty"`
		Author    string `json:"author,omitempty"`
		ThrowAway bool   `json:"throwaway,omitempty"`
	}

	var oid, oparent, ocreated string
	json.Unmarshal(*orig_v1["id"], &oid)
	json.Unmarshal(*orig_v1["parent"], &oparent)
	json.Unmarshal(*orig_v1["created"], &ocreated)
	v1 := v1comp{
		ID:      oid,
		Parent:  oparent,
		Created: ocreated,
		ContainerConfig: struct {
			Cmd []string
		}{
			Cmd: []string{""},
		},
	}
	v1_json, _ := json.Marshal(&v1)
	man_base.History[0].V1Compatibility = string(v1_json)

	//create new top layer V1Compatibility
	if _, ok := orig_v1["container"]; ok {
		delete(orig_v1, "container")
	}
	if _, ok := orig_v1["throwaway"]; ok {
		delete(orig_v1, "throwaway")
	}
	orig_v1["parent"] = orig_v1["id"]
	id, cerr := randomRead(32)
	if cerr != nil {
		return nil, cerr
	}

	sha_id, cerr := Sha256str(string(id))
	if cerr != nil {
		return nil, cerr
	}
	orig_v1["id"] = rawJson(sha_id)
	orig_v1["created"] = rawJson(time.Now().UTC().Format(time.RFC3339))
	//modify container_config & config
	if cconfig, ok := orig_v1["container_config"]; ok {
		cconfig_map := make(map[string]*json.RawMessage)
		err := json.Unmarshal([]byte(*cconfig), &cconfig_map)
		if err != nil {
			cerr := ErrNew(err, "unable to unmarshal container_config object")
			return nil, cerr
		}

		if _, ok := cconfig_map["Image"]; ok {
			if !strings.HasPrefix(layer_sha, "sha256:") {
				layer_sha = "sha256:" + layer_sha
			}
			cconfig_map["Image"] = rawJson(layer_sha)
		}

		//write back
		orig_v1["container_config"] = rawJson(cconfig_map)
	}

	if _, ok := orig_v1["config"]; ok {
		delete(orig_v1, "config")
		/**
		config_map := make(map[string]*json.RawMessage)
		err := json.Unmarshal([]byte(rawJsonToStr(config)), &config_map)
		if err != nil {
			cerr := ErrNew(err, "unable to unmarshal config object")
			return nil, cerr
		}
		if _, ok := config_map["Image"]; ok {
			if !strings.HasPrefix(layer_sha, "sha256:") {
				layer_sha = "sha256:" + layer_sha
			}
			config_map["Image"] = rawJson(layer_sha)
		}
		//write back
		orig_v1["config"] = rawJson(config_map)
		**/
	}

	histories := []schema1.History{schema1.History{V1Compatibility: rawJsonToStr(rawJson(orig_v1))}}
	histories = append(histories, man_base.History...)
	manifest.History = histories

	//modification ends
	pk, err := libtrust.GenerateECP256PrivateKey()
	if err != nil {
		cerr := ErrNew(err, "unexpected error generating private key")
		return nil, cerr
	}

	signedManifest, err := schema1.Sign(&manifest, pk)
	if err != nil {
		cerr := ErrNew(err, "error signning manifest")
		return nil, cerr
	}

	return signedManifest, nil
}

func rawJsonToStr(rjson *json.RawMessage) string {
	j, _ := json.Marshal(rjson)
	return string(j)
}

func rawJson(value interface{}) *json.RawMessage {
	jsonval, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return (*json.RawMessage)(&jsonval)
}

func randomRead(num int) ([]byte, *Error) {
	b := make([]byte, num)
	_, err := rand.Read(b)
	if err != nil {
		cerr := ErrNew(err, "error while generating random bytes")
		return nil, cerr
	}
	return b, nil
}

func UploadManifests(username, pass, name, tag, layer_sha, base_image string) *Error {
	log.SetOutput(ioutil.Discard)
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return cerr
	}

	signedManifest, serr := MakeManifestV1(hub, name, tag, layer_sha, base_image)
	if serr != nil {
		return serr
	}

	err = hub.PutManifest(name, tag, *signedManifest)
	if err != nil {
		cerr := ErrNew(err, "putting manifest error")
		return cerr
	}
	return nil
}

func UploadLayers(username, pass, name, tag, file, base_image string) (string, *Error) {
	log.SetOutput(ioutil.Discard)

	if !strings.Contains(name, "library/") && !strings.Contains(name, "/") {
		name = "library/" + name
	}

	sha256, err := Sha256file(file)
	if err != nil {
		return "", err
	}

	token, err := GetToken(name, username, pass, "pull,push")
	if err != nil {
		return "", err
	}

	hok, herr := HasBlob(name, token, sha256)
	if herr != nil {
		return "", herr
	}
	if !hok {
		//step 1: uploading blob
		_, err := UploadBlob(name, token, file)
		if err != nil {
			return "", err
		}

		//step 2: uploading new manifest
		err = UploadManifests(username, pass, name, tag, sha256, base_image)
		if err != nil {
			return "", err
		}
	} else {
		return sha256, nil
	}
	return sha256, nil
}

func DeleteManifest(username string, pass string, name string, tag string) *Error {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") && !strings.Contains(name, "/") {
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

//dir is temp dir used for tarball extraction, image dir is used for the storage of image data
func LoadDockerTar(dir, imagedir string) (string, map[string]int64, []string, *Error) {
	if !FolderExist(dir) {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s does not exist", dir))
		return "", nil, nil, cerr
	}

	var infos []DockerSaveInfo
	manifest_file := fmt.Sprintf("%s/manifest.json", dir)
	b, berr := ioutil.ReadFile(manifest_file)
	if berr != nil {
		cerr := ErrNew(berr, fmt.Sprintf("could not read file: %s", manifest_file))
		return "", nil, nil, cerr
	}

	jerr := json.Unmarshal(b, &infos)
	if jerr != nil {
		cerr := ErrNew(jerr, "could not unmarshal json bytes to objects")
		return "", nil, nil, cerr
	}

	name := infos[0].RepoTags[0]
	layer_data := make(map[string]int64)
	var layers []string
	for _, k := range infos[0].Layers {
		// k is sha256/layer.tar
		layer_path := fmt.Sprintf("%s/%s", dir, k)
		shavalue := filepath.Dir(k)
		//create tar.gz package of current tar package
		cerr := ConvertTar2Gzip(layer_path, fmt.Sprintf("%s/%s/%s", dir, shavalue, shavalue))
		if cerr != nil {
			return "", nil, nil, cerr
		}

		target_path := fmt.Sprintf("%s/%s.tar.gz", imagedir, shavalue)
		rerr := Rename(fmt.Sprintf("%s/%s/%s", dir, shavalue, shavalue), target_path)
		if rerr != nil {
			return "", nil, nil, rerr
		}
		file_length, ferr := GetFileLength(target_path)
		if ferr != nil {
			return "", nil, nil, ferr
		}
		layer_data[target_path] = file_length
		layers = append(layers, target_path)
	}

	return name, layer_data, layers, nil
}

func DownloadLayers(username string, pass string, name string, tag string, folder string) (map[string]int64, []string, *Error) {
	log.SetOutput(ioutil.Discard)
	if !strings.Contains(name, "library/") && !strings.Contains(name, "/") {
		name = "library/" + name
	}
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return nil, nil, err
		}
	}
	hub, err := registry.New(DOCKER_URL, username, pass)
	if err != nil {
		cerr := ErrNew(err, "create docker registry instance failure")
		return nil, nil, cerr
	}
	man, err := hub.ManifestV2(name, tag)
	if err != nil {
		cerr := ErrNew(err, "query docker manifest failure")
		return nil, nil, cerr
	}
	data := make(map[string]int64)
	var layer_order []string
	for _, element := range man.Layers {
		dig := element.Digest
		//reader, err := hub.DownloadLayer(name, dig)
		//function name is changed
		reader, err := hub.DownloadBlob(name, dig)
		if err != nil {
			cerr := ErrNew(err, "download docker layers failure")
			return nil, nil, cerr
		}
		defer reader.Close()
		if strings.HasSuffix(folder, "/") {
			folder = strings.TrimSuffix(folder, "/")
		}
		filename := folder + "/" + strings.TrimPrefix(dig.String(), "sha256:") + ".tar.gz"

		//if file exists, we skip to next file
		if FileExist(filename) {
			if size, err := GetFileSize(filename); err == nil {
				if size == element.Size {
					if _, ok := data[filename]; !ok {
						data[filename] = element.Size
						layer_order = append(layer_order, filename)
					}
					fmt.Println(fmt.Sprintf("File %s exists, skip...", filepath.Base(filename)))
					continue
				}
			}
		}

		//else not exist
		to, err := os.Create(filename)
		if err != nil {
			cerr := ErrNew(err, fmt.Sprintf("create file %s failure", filename))
			return nil, nil, cerr
		}
		defer to.Close()
		fmt.Println(fmt.Sprintf("Downloading file with type: %s, size: %d", element.MediaType, element.Size))

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
			return nil, nil, cerr
		}
		data[filename] = element.Size
		layer_order = append(layer_order, filename)
	}
	//data is map[string]int64, string is filename, int64 is the size of the layer
	//layer_order is the array of filenames
	return data, layer_order, nil
}

//find correct download url based on the yaml description file
func GenGithubURLfromYaml(name, tag, url, yaml, target_file string) (string, *Error) {
	if !FileExist(yaml) {
		//download ymal file first
		yaml_file_name := path.Base(yaml)
		yaml_folder_name := path.Dir(yaml)
		yaml_url := fmt.Sprintf("%s/%s", url, yaml_file_name)
		DirectDownloadFilefromGithub(yaml_file_name, yaml_url, yaml_folder_name)
	}

	ret_url := ""
	if FileExist(yaml) {
		_, config, lerr := LoadConfig(yaml)
		if lerr == nil {
			//assemble url
			name = strings.ToLower(name)
			tag = strings.ToLower(tag)
			if m, m_ok := config["map"].(map[string]interface{}); m_ok {
				if n, n_ok := m[name].(string); n_ok {
					name = n
				}
			}
			if v, v_ok := config["version"].(map[string]interface{}); v_ok {
				if v_data, v_data_ok := v[name]; v_data_ok {
					if v_array, v_array_ok := v_data.([]interface{}); v_array_ok {
						for _, arr := range v_array {
							ret, _ := CompareVersion(arr.(string), tag, ".")
							if ret == 0 || ret == -1 {
								tag = arr.(string)
								break
							}
						}
					}
				}
			}
		}
	}

	ret_url = fmt.Sprintf("%s/%s/%s/%s", url, name, tag, target_file)
	return ret_url, nil
}

//a patched version that could search file from different level and auto-select the closet one
//name: is the name of distro, tag: version of distro, filename: the target request file on github, url: fixed github prefix, folder: local folder saving downloaded file, yaml: yaml file saving distro info located on github
func DownloadFilefromGithubPlus(name, tag, filename, url, folder, yaml string) *Error {
	filepath := fmt.Sprintf("%s/%s", folder, filename)
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

	gen_url, _ := GenGithubURLfromYaml(name, tag, url, yaml, filename)

	derr := DirectDownloadFilefromGithub(filename, gen_url, folder)
	if derr != nil {
		//change to fallback search
		gen_url = fmt.Sprintf("%s/%s/%s", url, name, filename)
		derr = DirectDownloadFilefromGithub(filename, gen_url, folder)
		if derr != nil {
			gen_url = fmt.Sprintf("%s/default.%s", url, filename)
			derr = DirectDownloadFilefromGithub(filename, gen_url, folder)
			if derr != nil {
				return derr
			}
		}
	}

	fmt.Printf("Downloading %s from %s\n", filename, gen_url)
	return nil
}

//url is github url
func DirectDownloadFilefromGithub(filename string, url string, folder string) *Error {
	filepath := fmt.Sprintf("%s/%s", folder, filename)
	if !FolderExist(folder) {
		_, err := MakeDir(folder)
		if err != nil {
			return err
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", url))
		return cerr
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		cerr := ErrNew(ErrNExist, fmt.Sprintf("%s returns 404", url))
		return cerr
	}

	out, err := os.Create(filepath)
	if err != nil {
		cerr := ErrNew(ErrFileStat, fmt.Sprintf("%s file create error", filepath))
		return cerr
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", url, filepath))
		return cerr
	}
	return nil

}

func DownloadFilefromGithub(name string, tag string, filename string, url string, folder string) *Error {
	filepath := fmt.Sprintf("%s/%s", folder, filename)
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

	name = strings.ToLower(name)
	tag = strings.ToLower(tag)
	http_req := fmt.Sprintf("%s/%s/%s/%s", url, name, tag, filename)
	resp, err := http.Get(http_req)
	if err != nil {
		cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
		return cerr
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		http_req := fmt.Sprintf("%s/default.%s", url, filename)
		resp, err := http.Get(http_req)
		if err != nil {
			cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
			return cerr
		}
		defer resp.Body.Close()

		if resp.StatusCode == 404 {
			cerr := ErrNew(ErrHttpNotFound, fmt.Sprintf("http request to %s encounters failure", http_req))
			return cerr
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", http_req, filepath))
			return cerr
		}
		return nil
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		cerr := ErrNew(ErrFileIO, fmt.Sprintf("io copy from %s to %s encounters error", http_req, filepath))
		return cerr
	}
	return nil

}

type DockerHubToken struct {
	Token        string `json:"token"`
	Access_token string `json:"access_token"`
	Expire       int    `json:"expires_in"`
	Time         string `json:"issued_at"`
}

func GetToken(repository, username, password, action string) (string, *Error) {
	token_url := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:%s:%s", repository, action)
	client := &http.Client{}
	req, err := http.NewRequest("GET", token_url, nil)
	if err != nil {
		cerr := ErrNew(err, "could not create new http request")
		return "", cerr
	}
	if username != "" && password != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
		req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encoded))
	}
	rep, err := client.Do(req)
	if err != nil {
		cerr := ErrNew(err, "could not execute http client")
		return "", cerr
	}
	defer rep.Body.Close()
	body, err := ioutil.ReadAll(rep.Body)
	if err != nil {
		cerr := ErrNew(err, "could not read http response body")
		return "", cerr
	}
	var token DockerHubToken
	if err := json.Unmarshal(body, &token); err != nil {
		cerr := ErrNew(err, "could not UnmarshalJSON http response body")
		return "", cerr
	}
	return token.Token, nil
}

func UploadBlob(repository, token, file string) (bool, *Error) {
	initial_url := fmt.Sprintf("https://registry-1.docker.io/v2/%s/blobs/uploads/", repository)
	client := &http.Client{}
	req, err := http.NewRequest("POST", initial_url, nil)
	if err != nil {
		cerr := ErrNew(err, "could not create http request")
		return false, cerr
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	if err != nil {
		cerr := ErrNew(err, "could not execute http request")
		return false, cerr
	}
	defer resp.Body.Close()
	location := resp.Header.Get("Location")
	locationUrl, err := url.Parse(location)
	if err != nil {
		cerr := ErrNew(err, "could not parse location")
		return false, cerr
	}
	sha256, cerr := Sha256file(file)
	if cerr != nil {
		return false, cerr
	}
	data, err := os.Open(file)
	if err != nil {
		cerr := ErrNew(err, fmt.Sprintf("could not open file:%s", file))
		return false, cerr
	}

	upload_url := fmt.Sprintf("%s&digest=sha256:%s", locationUrl, sha256)
	req, err = http.NewRequest("PUT", upload_url, data)
	if err != nil {
		cerr := ErrNew(err, "could not create http request")
		return false, cerr
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/octet-stream")

	uresp, err := client.Do(req)
	if err != nil {
		cerr := ErrNew(err, "could not execute http request")
		return false, cerr
	}
	defer uresp.Body.Close()
	return true, nil
}

func HasBlob(repository, token, sha256 string) (bool, *Error) {
	checkUrl := fmt.Sprintf("https://registry-1.docker.io/v2/%s/blobs/sha256:%s", repository, sha256)
	client := &http.Client{}
	req, err := http.NewRequest("HEAD", checkUrl, nil)
	if err != nil {
		cerr := ErrNew(err, "could not create http request")
		return false, cerr
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)
	if err != nil {
		cerr := ErrNew(err, "could not execute http request")
		return false, cerr
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode == 200 {
		fmt.Println(resp)
		return true, nil
	}
	data, _ := ioutil.ReadAll(resp.Body)
	cerr := ErrNew(ErrHttpNotFound, string(data))
	return false, cerr
}

func DownloadBlob(repository, token, sha256 string) (io.ReadCloser, *Error) {
	download_url := fmt.Sprintf("https://registry-1.docker.io/v2/%s/blobs/sha256:%s", repository, sha256)
	req, err := http.NewRequest("GET", download_url, nil)
	if err != nil {
		cerr := ErrNew(err, "could not create http request")
		return nil, cerr
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	fmt.Println(resp)
	if err != nil {
		cerr := ErrNew(err, "could not execute http request")
		return nil, cerr
	}
	if resp != nil {
		return resp.Body, nil
	}
	cerr := ErrNew(ErrHttpNotFound, "unknown error of http request, no error while no data")
	return nil, cerr
}
