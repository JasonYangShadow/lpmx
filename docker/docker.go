package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/jasonyangshadow/lpmx/error"
	. "github.com/jasonyangshadow/lpmx/log"
)

//private funcs
func pullImage(name string, output string, auth bool) *Error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		cerr := ErrNew(err, "pullImage create env client encounters error")
		return cerr
	}
	if auth {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("Enter your username")
		user := scanner.Text()
		fmt.Println("Enter your password")
		pwd := scanner.Text()
		authConfig := types.AuthConfig{
			Username: user,
			Password: pwd,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			cerr := ErrNew(err, "pullImage json.Marshal encounters error")
			return cerr
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		out, err := cli.ImagePull(ctx, name, types.ImagePullOptions{RegistryAuth: authStr})
	} else {
		out, err := cli.ImagePull(ctx, name, types.ImagePullOptions{})
	}
	if err != nil {
		cerr := ErrNew(err, "pullImage image pull encounters error")
		return cerr
	}
	defer out.close()

	io.copy(output, out)
}

func PullImage(name string) *Error {
}
