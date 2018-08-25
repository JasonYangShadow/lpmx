package docker

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	. "github.com/jasonyangshadow/lpmx/error"
)

const (
	DOCKER_HUB_TOKEN = "https://auth.docker.io/token?serivce=registry.docker.io&scope="
	DOCKER_HUB_REQ   = "https://index.docker.io/v2/"
)

func createTransport() *http.Transport {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	return tr
}

func httpExCode(r *http.Response) (int, *Error) {
	switch r.StatusCode {
	case 200:
		return 200, nil
	case 401:
		cerr := ErrNew(ErrHttpUnauthorized, "401 unauthorized")
		return 401, cerr
	case 404:
		cerr := ErrNew(ErrHttpNotFound, "404 not found")
		return 404, cerr
	default:
		bodyBytes, _ := ioutil.ReadAll(r.Body)
		cerr := ErrNew(ErrUnknown, string(bodyBytes))
		return -1, cerr
	}
}

var http_client = &http.Client{Transport: createTransport(), Timeout: time.Second * 5}

func V2Available() (bool, *Error) {
	resp, err := http_client.Get(DOCKER_HUB_REQ)
	defer resp.Body.Close()
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		cerr := ErrNew(err, string(bodyBytes))
		return false, cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		return true, nil
	}
	return false, cerr
}

//return params are: successful(bool), statuscode, token, err(if it has)
func RegistryAuthenticate(name string, operations string) (bool, int, string, *Error) {
	requrl := DOCKER_HUB_TOKEN + "repository:" + name + ":" + operations
	resp, err := http_client.Get(requrl)
	defer resp.Body.Close()
	if err != nil {
		cerr := ErrNew(err, "http get encountered failure")
		return false, -1, "", cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 && cerr == nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		if len(bodyBytes) == 0 {
			cerr := ErrNew(ErrZero, "size of http return body is 0")
			return false, -1, "", cerr
		}
		data := make(map[string]interface{})
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			cerr := ErrNew(err, "json unmarshal http body failure")
			return false, -1, "", cerr
		}
		return true, 200, data["token"].(string), nil
	} else {
		cerr := ErrNew(err, "http get encountered failure")
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return false, resp.StatusCode, string(bodyBytes), cerr
	}
}

func RegistryAuthenticateBasic(name string, operations string, user string, pass string) (bool, int, string, *Error) {
	requrl := DOCKER_HUB_TOKEN + "repository:" + name + ":" + operations
	req, _ := http.NewRequest("GET", requrl, nil)
	auth := b64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	req.Header.Add("Authorization", "Basic "+auth)
	resp, err := http_client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		cerr := ErrNew(err, "http get encountered failure")
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return false, -1, string(bodyBytes), cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 && cerr == nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		if len(bodyBytes) == 0 {
			cerr := ErrNew(ErrZero, "size of http return body is 0")
			return false, -1, "", cerr
		}
		data := make(map[string]interface{})
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			cerr := ErrNew(err, "json unmarshal http body failure")
			return false, -1, "", cerr
		}
		return true, 200, data["token"].(string), nil
	} else {
		cerr := ErrNew(err, "http get encountered failure")
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return false, resp.StatusCode, string(bodyBytes), cerr
	}
}

func PullManifest(name string, tag string, token string) *Error {
	requrl := DOCKER_HUB_REQ + name + "/manifests/" + tag
	req, _ := http.NewRequest("Get", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	fmt.Println(resp.Header)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		cerr := ErrNew(err, string(bodyBytes))
		return cerr
	}

	fmt.Println(string(bodyBytes))
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		data := make(map[string]interface{})
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			cerr := ErrNew(err, "json unmarshal http body failure")
			return cerr
		}
		fmt.Println(data)
		return nil
	}
	return cerr
}

func CheckManifest(name string, tag string, token string) (bool, *Error) {
	requrl := DOCKER_HUB_REQ + name + "/manifests/" + tag
	req, _ := http.NewRequest("HEAD", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		cerr := ErrNew(err, string(bodyBytes))
		return false, cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		return true, nil
	}
	return false, cerr
}

func CheckLayers(name string, digest string, token string) (bool, *Error) {
	requrl := DOCKER_HUB_REQ + name + "/blobs/" + digest
	req, _ := http.NewRequest("HEAD", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		cerr := ErrNew(err, string(bodyBytes))
		return false, cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		return true, nil
	}
	return false, cerr
}

func PushLayer(name string, token string) *Error {
	requrl := DOCKER_HUB_REQ + name + "/blobs/uploads/"
	req, _ := http.NewRequest("POST", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		cerr := ErrNew(err, string(bodyBytes))
		return cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		return nil
	}
	return cerr
}

func PullLayer(name string, digest string, path string, token string) *Error {
	requrl := DOCKER_HUB_REQ + name + "/blobs/" + digest
	req, _ := http.NewRequest("GET", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		cerr := ErrNew(err, string(bodyBytes))
		return cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		out, err := os.Create(path)
		if err != nil {
			cerr := ErrNew(ErrUnknown, "create file error")
			return cerr
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			cerr := ErrNew(ErrUnknown, fmt.Sprintf("can't copy file from http to %s", path))
			return cerr
		}
		return nil
	}
	return cerr
}

func ListRepositories(token string) (string, *Error) {
	requrl := DOCKER_HUB_REQ + "_catalog"
	req, _ := http.NewRequest("GET", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	defer resp.Body.Close()
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		cerr := ErrNew(err, string(bodyBytes))
		return "", cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		return string(bodyBytes), nil
	}
	return string(bodyBytes), cerr
}

func ListTags(name string, token string) (string, *Error) {
	requrl := DOCKER_HUB_REQ + name + "/tags/list"
	req, _ := http.NewRequest("GET", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		cerr := ErrNew(err, string(bodyBytes))
		return "", cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		return string(bodyBytes), nil
	}
	return string(bodyBytes), cerr
}

func GetCatalog(token string) (string, *Error) {
	requrl := DOCKER_HUB_REQ + "_catalog"
	req, _ := http.NewRequest("GET", requrl, nil)
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http_client.Do(req)
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		cerr := ErrNew(err, string(bodyBytes))
		return "", cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 {
		return string(bodyBytes), nil
	}
	return string(bodyBytes), cerr
}
