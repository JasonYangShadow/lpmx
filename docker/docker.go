package docker

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	. "github.com/jasonyangshadow/lpmx/error"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	DOCKER_HUB_TOKEN = "https://auth.docker.io/token?serivce=registry.docker.io&scope="
	CATALOG_LIMIT    = 50
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
		cerr := ErrNew(ErrUnknown, "http exception unknown")
		return -1, cerr
	}
}

var http_client = &http.Client{Transport: createTransport()}

//return params are: successful(bool), statuscode, token, refresh_token, err(if it has)
func RegistryAuthenticate(target string, operations string) (bool, int, string, *Error) {
	requrl := DOCKER_HUB_TOKEN + "repository:" + target + ":" + operations
	resp, err := http_client.Get(requrl)
	defer resp.Body.Close()
	if err != nil {
		cerr := ErrNew(err, "http get encountered failure")
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return false, -1, string(bodyBytes), cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 && cerr == nil {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
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

func RegistryAuthenticateBasic(target string, operations string, user string, pass string) (bool, int, string, *Error) {
	requrl := DOCKER_HUB_TOKEN + "repository:" + target + ":" + operations
	req, _ := http.NewRequest("GET", requrl, nil)
	auth := b64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	req.Header.Add("Authorization", auth)
	resp, err := http_client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		cerr := ErrNew(err, "http get encountered failure")
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return false, -1, string(bodyBytes), cerr
	}
	retcode, cerr := httpExCode(resp)
	if retcode == 200 && cerr == nil {
		return true, 200, "token", nil
	} else {
		cerr := ErrNew(err, "http get encountered failure")
		fmt.Println(resp)
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return false, resp.StatusCode, string(bodyBytes), cerr
	}
}
