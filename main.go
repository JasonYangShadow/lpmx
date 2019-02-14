package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/jasonyangshadow/lpmx/container"
	. "github.com/jasonyangshadow/lpmx/log"
	. "github.com/jasonyangshadow/lpmx/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	checklist      = []string{"faked-sysv", "libevent", "libfakechroot.so", "libfakeroot.so", "libmemcached", "libsasl2", "memcached", ".info"}
	bool_checklist = make([]bool, len(checklist))
)

func checkCompleteness() bool {
	dir, err := GetCurrDir()
	if err != nil {
		LOGGER.Fatal(err.Error())
		return false
	}

	filepath.Walk(fmt.Sprintf("%s/.lpmxsys", dir), func(path string, info os.FileInfo, err error) error {
		for idx, item := range checklist {
			if strings.HasPrefix(info.Name(), item) {
				if item != ".info" {
					CheckFilePermission(path, 0755, true)
				}
				bool_checklist[idx] = true
			}
		}
		return nil
	})

	for idx, item := range bool_checklist {
		if !item {
			LOGGER.Fatal(fmt.Sprintf("necessary file %s does not exist, you may need to use './lpmx init' to initialize lpmx iteself firstly", checklist[idx]))
			return false
		}
	}
	return true
}

func main() {
	var InitReset bool
	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "init the lpmx itself",
		Long:  "init command is the basic command of lpmx, which is used for initializing lpmx system",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Init(InitReset)
			if err != nil {
				LOGGER.Fatal(err.Error())
			}
			checkCompleteness()
		},
	}
	initCmd.Flags().BoolVarP(&InitReset, "reset", "r", false, "initialize by force(optional)")

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "list the containers in lpmx system",
		Long:  "list command is the basic command of lpmx, which is used for listing all the containers registered",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := List()
			if err != nil {
				LOGGER.Fatal(err.Error())
			}
		},
	}

	var RunSource string
	var RunConfig string
	var RunPassive bool
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "run container based on specific directory",
		Long:  "run command is the basic command of lpmx, which is used for initializing, creating and running container based on specific directory",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			RunSource, _ = filepath.Abs(RunSource)
			if RunConfig != "" {
				RunConfig, _ = filepath.Abs(RunConfig)
			} else {
				config_path := fmt.Sprintf("%s/setting.yml", RunSource)
				if FileExist(config_path) {
					RunConfig = config_path
				} else {
					LOGGER.Fatal("can't locate the setting.yml in source folder")
				}
			}
			configmap := make(map[string]interface{})
			configmap["dir"] = RunSource
			configmap["config"] = RunConfig
			configmap["passive"] = RunPassive
			err := Run(&configmap)
			if err != nil {
				LOGGER.Fatal(err.Error())
			}
		},
	}
	runCmd.Flags().StringVarP(&RunSource, "source", "s", "", "required")
	runCmd.MarkFlagRequired("source")
	runCmd.Flags().StringVarP(&RunConfig, "config", "c", "", "optional(if the setting.yml exists in source folder, then you don't need to specify the path)")
	runCmd.Flags().BoolVarP(&RunPassive, "passive", "p", false, "optional")

	var GetId string
	var GetName string
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "get settings from memcache server",
		Long:  "get command is the basic command of lpmx, which is used for getting settings from cache server",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := Get(GetId, GetName)
			if err != nil {
				LOGGER.Fatal(err.Error())
			}
		},
	}
	getCmd.Flags().StringVarP(&GetId, "id", "i", "", "required")
	getCmd.MarkFlagRequired("id")
	getCmd.Flags().StringVarP(&GetName, "name", "n", "", "required")
	getCmd.MarkFlagRequired("name")

	var RExecIp string
	var RExecPort string
	var RExecTimeout string
	var rpcExecCmd = &cobra.Command{
		Use:   "exec",
		Short: "exec command remotely",
		Long:  "rpc exec sub-command is the advanced comand of lpmx, which is used for executing command remotely through rpc",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			_, err := RPCExec(RExecIp, RExecPort, RExecTimeout, args[0], args[1:]...)
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				LOGGER.Info("DONE")
			}
		},
	}
	rpcExecCmd.Flags().StringVarP(&RExecIp, "ip", "i", "", "required")
	rpcExecCmd.MarkFlagRequired("ip")
	rpcExecCmd.Flags().StringVarP(&RExecPort, "port", "p", "", "required")
	rpcExecCmd.MarkFlagRequired("port")
	rpcExecCmd.Flags().StringVarP(&RExecTimeout, "timeout", "t", "", "optional")

	var RQueryIp string
	var RQueryPort string
	var rpcQueryCmd = &cobra.Command{
		Use:   "query",
		Short: "query the information of commands executed remotely",
		Long:  "rpc query sub-command is the advanced comand of lpmx, which is used for querying the information of commands executed remotely through rpc",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			res, err := RPCQuery(RQueryIp, RQueryPort)
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				fmt.Println("PID", "CMD")
				for k, v := range res.RPCMap {
					fmt.Println(k, v)
				}
			}
		},
	}
	rpcQueryCmd.Flags().StringVarP(&RQueryIp, "ip", "i", "", "required")
	rpcQueryCmd.MarkFlagRequired("ip")
	rpcQueryCmd.Flags().StringVarP(&RQueryPort, "port", "p", "", "required")
	rpcQueryCmd.MarkFlagRequired("port")

	var RDeleteIp string
	var RDeletePort string
	var RDeletePid string
	var rpcDeleteCmd = &cobra.Command{
		Use:   "kill",
		Short: "kill the commands executed remotely via pid",
		Long:  "rpc delete sub-command is the advanced comand of lpmx, which is used for killing the commands executed remotely through rpc via pid",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			i, aerr := strconv.Atoi(RDeletePid)
			if aerr != nil {
				LOGGER.Fatal(aerr.Error())
			}
			_, err := RPCDelete(RDeleteIp, RDeletePort, i)
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				LOGGER.Info("DONE")
			}
		},
	}
	rpcDeleteCmd.Flags().StringVarP(&RDeleteIp, "ip", "i", "", "required")
	rpcDeleteCmd.MarkFlagRequired("ip")
	rpcDeleteCmd.Flags().StringVarP(&RDeletePort, "port", "p", "", "required")
	rpcDeleteCmd.MarkFlagRequired("port")
	rpcDeleteCmd.Flags().StringVarP(&RDeletePid, "pid", "d", "", "required")
	rpcDeleteCmd.MarkFlagRequired("pid")

	var rpcCmd = &cobra.Command{
		Use:   "rpc",
		Short: "exec command remotely",
		Long:  "rpc command is the advanced comand of lpmx, which is used for executing command remotely through rpc",
	}
	rpcCmd.AddCommand(rpcExecCmd, rpcQueryCmd, rpcDeleteCmd)

	//docker cmd
	var DockerDownloadUser string
	var DockerDownloadPass string
	var dockerDownloadCmd = &cobra.Command{
		Use:   "download",
		Short: "download the docker images from docker hub",
		Long:  "docker download sub-command is the advanced command of lpmx, which is used for downloading the images from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerDownload(args[0], DockerDownloadUser, DockerDownloadPass)
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				LOGGER.Info("DONE")
			}
		},
	}
	dockerDownloadCmd.Flags().StringVarP(&DockerDownloadUser, "user", "u", "", "optional")
	dockerDownloadCmd.Flags().StringVarP(&DockerDownloadPass, "pass", "p", "", "optional")

	var DockerCreateName string
	var dockerCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "initialize the local docker images",
		Long:  "docker create sub-command is the advanced command of lpmx, which is used for initializing and running the images downloaded from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			err := DockerCreate(args[0], DockerCreateName)
			if err != nil {
				LOGGER.Fatal(err.Error())
			}
		},
	}
	dockerCreateCmd.Flags().StringVarP(&DockerCreateName, "name", "n", "", "optional")

	var dockerDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "delete the local docker images",
		Long:  "docker delete sub-command is the advanced command of lpmx, which is used for deleting the images downloaded from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerDelete(args[0])
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				LOGGER.Info("DONE")
			}
		},
	}

	var dockerSearchCmd = &cobra.Command{
		Use:   "search",
		Short: "search the docker images from docker hub",
		Long:  "docker search sub-command is the advanced command of lpmx, which is used for searching the images from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			tags, err := DockerSearch(args[0])
			if err != nil {
				LOGGER.Error(err.Error())
			}
			fmt.Println(fmt.Sprintf("Name: %s, Available Tags: %s", args[0], tags))
		},
	}

	var dockerListCmd = &cobra.Command{
		Use:   "list",
		Short: "list local docker images",
		Long:  "docker list sub-command is the advanced command of lpmx, which is used for listing local images downloaded from docker hub",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerList()
			if err != nil {
				LOGGER.Error(err.Error())
			}
		},
	}

	var dockerResetCmd = &cobra.Command{
		Use:   "reset",
		Short: "reset local docker base layers",
		Long:  "docker reset sub-command is the advanced command of lpmx, which is used for clearing current extacted base layers and reextracting them.(Only for Advanced Use)",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerReset(args[0])
			if err != nil {
				LOGGER.Error(err.Error())
			} else {
				LOGGER.Info("DONE")
			}
		},
	}

	var DockerPushUser string
	var DockerPushPass string
	var DockerPushName string
	var DockerPushTag string
	var DockerPushId string
	var dockerPushCmd = &cobra.Command{
		Use:   "push",
		Short: "push local fake unionfs layer to dockerhub",
		Long:  "docker push sub-command is the advanced command of lpmx, which is used for pacaking and pushing current rw layer to dockerhub.",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerPush(DockerPushUser, DockerPushPass, DockerPushName, DockerPushPass, DockerPushId)
			if err != nil {
				LOGGER.Error(err.Error())
			} else {
				LOGGER.Info("DONE")
			}
		},
	}
	dockerPushCmd.Flags().StringVarP(&DockerPushUser, "user", "u", "", "optional")
	dockerPushCmd.Flags().StringVarP(&DockerPushPass, "pass", "p", "", "optional")
	dockerPushCmd.Flags().StringVarP(&DockerPushName, "name", "n", "", "required")
	dockerPushCmd.MarkFlagRequired("name")
	dockerPushCmd.Flags().StringVarP(&DockerPushTag, "tag", "t", "", "required")
	dockerPushCmd.MarkFlagRequired("tag")
	dockerPushCmd.Flags().StringVarP(&DockerPushId, "id", "i", "", "required")
	dockerPushCmd.MarkFlagRequired("id")

	var dockerCmd = &cobra.Command{
		Use:   "docker",
		Short: "docker command",
		Long:  "docker command is the advanced comand of lpmx, which is used for executing docker related commands",
	}
	dockerCmd.AddCommand(dockerCreateCmd, dockerSearchCmd, dockerListCmd, dockerDeleteCmd, dockerDownloadCmd, dockerResetCmd, dockerPushCmd)

	var ExposeId string
	var ExposeName string
	var exposeCmd = &cobra.Command{
		Use:   "expose",
		Short: "expose program inside container",
		Long:  "expose command is the advanced command of lpmx, which is used for exposing binaries inside containers to host",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := Expose(ExposeId, ExposeName)
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				LOGGER.Info("DONE")
			}
		},
	}
	exposeCmd.Flags().StringVarP(&ExposeId, "id", "i", "", "required")
	exposeCmd.MarkFlagRequired("id")
	exposeCmd.Flags().StringVarP(&ExposeName, "name", "n", "", "required")
	exposeCmd.MarkFlagRequired("name")

	var resumeCmd = &cobra.Command{
		Use:   "resume",
		Short: "resume the registered container",
		Long:  "resume command is the basic command of lpmx, which is used for resuming the registered container via id",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			err := Resume(args[0], args[1:]...)
			if err != nil {
				LOGGER.Fatal(err.Error())
			}
		},
	}

	var destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy the registered container",
		Long:  "destroy command is the basic command of lpmx, which is used for destroying the registered container via id",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := Destroy(args[0])
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				LOGGER.WithFields(logrus.Fields{
					"container id": args[0],
				}).Info("container is destroyed")
			}
		},
	}

	var SetId string
	var SetType string
	var SetProg string
	var SetVal string
	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "set environment variables for container",
		Long:  "set command is an advanced comand of lpmx, which is used for setting environment variables of running containers, you should clearly know what you want before using this command, it will reduce the performance heavily",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			bcheck := checkCompleteness()
			if !bcheck {
				return
			}
			err := CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			if !strings.Contains(SetVal, ":") {
				LOGGER.WithFields(logrus.Fields{
					"value": SetVal,
				}).Fatal("the program value you input does not have ':', the format should be 'file1:replace_file1;file2:replace_file2'")
				return
			}
			err := Set(SetId, SetType, SetProg, SetVal)
			if err != nil {
				LOGGER.Fatal(err.Error())
			} else {
				LOGGER.WithFields(logrus.Fields{
					"container id": SetId,
				}).Info("container is set with new environment variables")
			}
		},
	}
	setCmd.Flags().StringVarP(&SetId, "id", "i", "", "required(container id, you can get the id by command 'lpmx list')")
	setCmd.MarkFlagRequired("id")
	setCmd.Flags().StringVarP(&SetType, "type", "t", "", "required('add_map','remove_map')")
	setCmd.MarkFlagRequired("type")
	setCmd.Flags().StringVarP(&SetProg, "name", "n", "", "required(should be the name of libc 'system calls wrapper')")
	setCmd.MarkFlagRequired("name")
	setCmd.Flags().StringVarP(&SetVal, "value", "v", "", "required(value(file1:replace_file1;file2:repalce_file2;)) ")
	setCmd.MarkFlagRequired("value")

	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx rootless container",
	}
	rootCmd.AddCommand(initCmd, destroyCmd, listCmd, runCmd, setCmd, resumeCmd, rpcCmd, getCmd, dockerCmd, exposeCmd)
	rootCmd.Execute()
}
