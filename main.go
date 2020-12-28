package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/JasonYangShadow/lpmx/container"
	. "github.com/JasonYangShadow/lpmx/error"
	. "github.com/JasonYangShadow/lpmx/log"
	. "github.com/JasonYangShadow/lpmx/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	checklist = []string{"faked-sysv", "libfakechroot.so", "libfakeroot.so", "libmemcached"}
)

const (
	VERSION = "alpha-1.7"
)

func checkCompleteness() *Error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	_, err = WalkandCheckFilePermission(fmt.Sprintf("%s/.lpmxsys", dir), checklist, 0755, true)
	if err != nil {
		return err
	}

	err = CheckCompleteness(fmt.Sprintf("%s/.lpmxsys", dir), []string{".info"})
	if err != nil {
		return err
	}

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		LOGGER.Debug(fmt.Sprintf("%s = %s", pair[0], pair[1]))
	}
	return nil
}

func main() {
	var InitReset bool
	var InitDep string
	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "init the lpmx itself",
		Long:  "init command is the basic command of lpmx, which is used for initializing lpmx system",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Init(InitReset, InitDep)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}
	initCmd.Flags().BoolVarP(&InitReset, "reset", "r", false, "initialize by force(optional)")
	initCmd.Flags().StringVarP(&InitDep, "dependency", "d", "", "dependency tar ball(optional)")

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "list the containers in lpmx system",
		Long:  "list command is the basic command of lpmx, which is used for listing all the containers registered",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := List()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}

	var GetId string
	var GetName string
	var GetMode bool
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "get settings from memcache server",
		Long:  "get command is the basic command of lpmx, which is used for getting settings from cache server",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}

			err = CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := Get(GetId, GetName, GetMode)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}
	getCmd.Flags().StringVarP(&GetId, "id", "i", "", "required")
	getCmd.MarkFlagRequired("id")
	getCmd.Flags().StringVarP(&GetName, "name", "n", "", "required")
	getCmd.MarkFlagRequired("name")
	getCmd.Flags().BoolVarP(&GetMode, "filecache", "c", false, "use FileCaache rather than Memcached(optional)")

	var DownloadSource string
	var downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "download files from online storage",
		Long:  "download command is the basic command of lpmx, which is used for downloading dependency or other files from online storage",
		Args:  cobra.ExactArgs(2),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			if DownloadSource == "gdrive" {
				new_url, nerr := GetGDriveDownloadLink(args[0])
				if nerr != nil {
					LOGGER.Fatal(nerr.Error())
					return
				}
				target_folder := filepath.Dir(args[1])
				target_file := filepath.Base(args[1])
				derr := DownloadFile(new_url, target_folder, target_file)
				if derr != nil {
					LOGGER.Fatal(derr.Error())
					return
				}

				LOGGER.Info("DONE")
				return
			}
		},
	}
	downloadCmd.Flags().StringVarP(&DownloadSource, "source", "s", "", "required, download source(gdrive)")
	downloadCmd.MarkFlagRequired("source")

	var RExecIp string
	var RExecPort string
	var RExecTimeout string
	var rpcExecCmd = &cobra.Command{
		Use:   "exec",
		Short: "exec command remotely",
		Long:  "rpc exec sub-command is the advanced comand of lpmx, which is used for executing command remotely through rpc",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = CheckAndStartMemcache()
			if err != nil && err.Err != ErrNExist {
				LOGGER.Fatal(err.Error())
				return
			}

			if err != nil && err.Err == ErrNExist {
				LOGGER.Warn("memcached related components are missing, functions may not work properly")
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			_, err := RPCExec(RExecIp, RExecPort, RExecTimeout, args[0], args[1:]...)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
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
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			res, err := RPCQuery(RQueryIp, RQueryPort)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				fmt.Println("PID", "CMD")
				for k, v := range res.RPCMap {
					fmt.Println(k, v)
				}
				return
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
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			i, aerr := strconv.Atoi(RDeletePid)
			if aerr != nil {
				LOGGER.Fatal(aerr.Error())
				return
			}
			_, err := RPCDelete(RDeleteIp, RDeletePort, i)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
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
		Long:  "rpc command is one advanced comand of lpmx, which is used for executing command remotely through rpc",
	}
	rpcCmd.AddCommand(rpcExecCmd, rpcQueryCmd, rpcDeleteCmd)

	//docker cmd
	var DockerDownloadUser string
	var DockerDownloadPass string
	var DockerDownloadMerge bool
	var dockerDownloadCmd = &cobra.Command{
		Use:   "download",
		Short: "download the docker images from docker hub",
		Long:  "docker download sub-command is one advanced command of lpmx, which is used for downloading the images from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			var err *Error
			LOGGER.Info(fmt.Sprintf("Start downloading %s", args[0]))
			err = DockerDownload(args[0], DockerDownloadUser, DockerDownloadPass)
			if err != nil && err.Err != ErrExist {
				LOGGER.Fatal(err.Error())
				return
			}
			if DockerDownloadMerge {
				//then create merged image secondly
				LOGGER.Info(fmt.Sprintf("Start merging %s", args[0]))
				err = DockerMerge(args[0], DockerDownloadUser, DockerDownloadPass)
			}
			if err != nil && err != ErrExist {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}
	dockerDownloadCmd.Flags().BoolVarP(&DockerDownloadMerge, "merge", "m", false, "merge all layers(optional)")
	dockerDownloadCmd.Flags().StringVarP(&DockerDownloadUser, "user", "u", "", "optional")
	dockerDownloadCmd.Flags().StringVarP(&DockerDownloadPass, "pass", "p", "", "optional")

	var dockerAddCmd = &cobra.Command{
		Use:   "add",
		Short: "add the exported image to system",
		Long:  "docker add sub-command is one advanced command of lpmx, which is used for adding packaged docker image to lpmx system",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerAdd(args[0])
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}

	var DockerPackageUser string
	var DockerPackagePass string
	var dockerPackageCmd = &cobra.Command{
		Use:   "package",
		Short: "package the docker images from docker hub for offline usage",
		Long:  "docker package sub-command is the advanced command of lpmx, which is used for packaging the images downloaded from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerPackage(args[0], DockerPackageUser, DockerPackagePass)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info(fmt.Sprintf("%s.tar.gz locates inside 'package' folder", args[0]))
				return
			}
		},
	}
	dockerPackageCmd.Flags().StringVarP(&DockerPackageUser, "user", "u", "", "optional")
	dockerPackageCmd.Flags().StringVarP(&DockerPackagePass, "pass", "p", "", "optional")

	var DockerCommitId string
	var DockerCommitName string
	var DockerCommitTag string
	var dockerCommitCmd = &cobra.Command{
		Use:   "commit",
		Short: "commit docker container",
		Long:  "docker commit sub-command is the advanced command of lpmx, which is used for committing container to new image",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerCommit(DockerCommitId, DockerCommitName, DockerCommitTag)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}
	dockerCommitCmd.Flags().StringVarP(&DockerCommitId, "id", "i", "", "required")
	dockerCommitCmd.MarkFlagRequired("id")
	dockerCommitCmd.Flags().StringVarP(&DockerCommitName, "name", "n", "", "required")
	dockerCommitCmd.MarkFlagRequired("name")
	dockerCommitCmd.Flags().StringVarP(&DockerCommitTag, "tag", "t", "", "required")
	dockerCommitCmd.MarkFlagRequired("tag")

	var DockerCreateName string
	var DockerCreateVolume string
	var DockerCreateEngine string
	var dockerCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "initialize the local docker images",
		Long:  "docker create sub-command is the advanced command of lpmx, which is used for initializing and running the images downloaded from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = CheckAndStartMemcache()
			if err != nil && err.Err != ErrNExist {
				LOGGER.Fatal(err.Error())
				return
			}

			if err != nil && err.Err == ErrNExist {
				LOGGER.Warn("memcached related components are missing, functions may not work properly")
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			err := CommonCreate(args[0], DockerCreateName, DockerCreateVolume, DockerCreateEngine)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}
	dockerCreateCmd.Flags().StringVarP(&DockerCreateName, "name", "n", "", "optional")
	dockerCreateCmd.Flags().StringVarP(&DockerCreateVolume, "volume", "v", "", "optional")
	dockerCreateCmd.Flags().StringVarP(&DockerCreateEngine, "engine", "e", "", "use engine(optional)")

	var DockerRunVolume string
	var DockerRunMode string
	var dockerRunCmd = &cobra.Command{
		Use:   "fastrun",
		Short: "run container in a fast way without switching into shell",
		Long:  "docker run sub-command is the advanced command of lpmx, which is used for fast running the container created from Docker image",
		Args:  cobra.ExactArgs(2),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = CheckAndStartMemcache()
			if err != nil && err.Err != ErrNExist {
				LOGGER.Fatal(err.Error())
				return
			}
			if err != nil && err.Err == ErrNExist {
				LOGGER.Warn("memcached related components are missing, functions may not work properly")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := CommonFastRun(args[0], DockerRunVolume, args[1], DockerRunMode)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}
	dockerRunCmd.Flags().StringVarP(&DockerRunVolume, "volume", "v", "", "optional")
	dockerRunCmd.Flags().StringVarP(&DockerRunMode, "engine", "e", "", "use engine(optional)")

	var DockerDeletePermernant bool
	var dockerDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "delete the local docker images",
		Long:  "docker delete sub-command is the advanced command of lpmx, which is used for deleting the images downloaded from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := CommonDelete(args[0], DockerDeletePermernant)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}
	dockerDeleteCmd.Flags().BoolVarP(&DockerDeletePermernant, "permernant", "p", false, "permernantly delete all layers of the target image(optional)")

	var dockerSearchCmd = &cobra.Command{
		Use:   "search",
		Short: "search the docker images from docker hub",
		Long:  "docker search sub-command is the advanced command of lpmx, which is used for searching the images from docker hub",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			tags, err := DockerSearch(args[0])
			if err != nil {
				LOGGER.Error(err.Error())
				return
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
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := CommonList("Docker")
			if err != nil {
				LOGGER.Error(err.Error())
				return
			}
		},
	}

	var dockerResetCmd = &cobra.Command{
		Use:   "reset",
		Short: "reset local docker base layers",
		Long:  "docker reset sub-command is the advanced command of lpmx, which is used for clearing current extacted base layers and reextracting them.(Only for Advanced Use)",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerReset(args[0])
			if err != nil {
				LOGGER.Error(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
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
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerPush(DockerPushUser, DockerPushPass, DockerPushName, DockerPushPass, DockerPushId)
			if err != nil {
				LOGGER.Error(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
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

	var dockerLoadCmd = &cobra.Command{
		Use:   "load",
		Short: "load the 'docker save' generated tar ball to system",
		Long:  "docker load sub-command is one advanced command of lpmx, which is used for importing 'docker save' generated tar ball to system",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := DockerLoad(args[0])
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}

	var dockerCmd = &cobra.Command{
		Use:   "docker",
		Short: "docker command",
		Long:  "docker command is the advanced command of lpmx, which is used for executing docker related commands",
	}
	dockerCmd.AddCommand(dockerCreateCmd, dockerSearchCmd, dockerListCmd, dockerDeleteCmd, dockerDownloadCmd, dockerResetCmd, dockerPackageCmd, dockerAddCmd, dockerCommitCmd, dockerLoadCmd, dockerRunCmd)

	var SingularityLoadName string
	var SingularityLoadTag string
	var singularityLoadCmd = &cobra.Command{
		Use:   "load",
		Short: "load local sif image",
		Long:  "singularity load sub-command is the advanced command of lpmx, which is used for loading sif image.",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := SingularityLoad(args[0], SingularityLoadName, SingularityLoadTag)
			if err != nil && err != ErrExist {
				LOGGER.Error(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}
	singularityLoadCmd.Flags().StringVarP(&SingularityLoadName, "name", "n", "", "required")
	singularityLoadCmd.MarkFlagRequired("name")
	singularityLoadCmd.Flags().StringVarP(&SingularityLoadTag, "tag", "t", "", "required")
	singularityLoadCmd.MarkFlagRequired("tag")

	var SingularityCreateName string
	var SingularityCreateVolume string
	var SingularityCreateEngine string
	var singularityCreateCmd = &cobra.Command{
		Use:   "create",
		Short: "initialize the local singularity images",
		Long:  "singularity create sub-command is the advanced command of lpmx, which is used for initializing and running the sif image",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = CheckAndStartMemcache()
			if err != nil && err.Err != ErrNExist {
				LOGGER.Fatal(err.Error())
				return
			}

			if err != nil && err.Err == ErrNExist {
				LOGGER.Warn("memcached related components are missing, functions may not work properly")
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			err := CommonCreate(args[0], SingularityCreateName, SingularityCreateVolume, SingularityCreateEngine)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}
	singularityCreateCmd.Flags().StringVarP(&SingularityCreateName, "name", "n", "", "optional")
	singularityCreateCmd.Flags().StringVarP(&SingularityCreateVolume, "volume", "v", "", "optional")
	singularityCreateCmd.Flags().StringVarP(&SingularityCreateEngine, "engine", "e", "", "use engine(optional)")

	var SingularityDeletePermernant bool
	var singularityDeleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "delete the local singularity images",
		Long:  "singularity delete sub-command is the advanced command of lpmx, which is used for deleting the singularity image",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := CommonDelete(args[0], SingularityDeletePermernant)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}
	singularityDeleteCmd.Flags().BoolVarP(&SingularityDeletePermernant, "permernant", "p", false, "permernantly delete all layers of the target image(optional)")

	var singularityListCmd = &cobra.Command{
		Use:   "list",
		Short: "list local singularity images",
		Long:  "singularity list sub-command is the advanced command of lpmx, which is used for listing local singularity images",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := CommonList("Singularity")
			if err != nil {
				LOGGER.Error(err.Error())
				return
			}
		},
	}

	var SingularityRunVolume string
	var SingularityRunMode string
	var singularityRunCmd = &cobra.Command{
		Use:   "fastrun",
		Short: "run container in a fast way without switching into shell",
		Long:  "singularity run sub-command is the advanced command of lpmx, which is used for fast running the container created from singularity image",
		Args:  cobra.ExactArgs(2),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = CheckAndStartMemcache()
			if err != nil && err.Err != ErrNExist {
				LOGGER.Fatal(err.Error())
				return
			}
			if err != nil && err.Err == ErrNExist {
				LOGGER.Warn("memcached related components are missing, functions may not work properly")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := CommonFastRun(args[0], DockerRunVolume, args[1], SingularityRunMode)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}
	singularityRunCmd.Flags().StringVarP(&SingularityRunVolume, "volume", "v", "", "optional")
	singularityRunCmd.Flags().StringVarP(&SingularityRunMode, "engine", "e", "", "use engine(optional)")

	var singularityCmd = &cobra.Command{
		Use:   "singularity",
		Short: "singularity command",
		Long:  "singularity command is the advanced command of lpmx, which is used for executing singularity related commands",
	}
	singularityCmd.AddCommand(singularityLoadCmd, singularityCreateCmd, singularityDeleteCmd, singularityListCmd, singularityRunCmd)

	var ExposeId string
	var ExposePath string
	var ExposeName string
	var exposeCmd = &cobra.Command{
		Use:   "expose",
		Short: "expose program inside container",
		Long:  "expose command is the advanced command of lpmx, which is used for exposing binaries inside containers to host",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := Expose(ExposeId, ExposePath, ExposeName)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
				return
			}
		},
	}
	exposeCmd.Flags().StringVarP(&ExposeId, "id", "i", "", "required")
	exposeCmd.MarkFlagRequired("id")
	exposeCmd.Flags().StringVarP(&ExposePath, "path", "p", "", "required")
	exposeCmd.MarkFlagRequired("path")
	exposeCmd.Flags().StringVarP(&ExposeName, "name", "n", "", "required")
	exposeCmd.MarkFlagRequired("name")

	var ResumeEngine bool
	var resumeCmd = &cobra.Command{
		Use:   "resume",
		Short: "resume the registered container",
		Long:  "resume command is the basic command of lpmx, which is used for resuming the registered container via id",
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = CheckAndStartMemcache()
			if err != nil && err.Err != ErrNExist {
				LOGGER.Fatal(err.Error())
				return
			}

			if err != nil && err.Err == ErrNExist {
				LOGGER.Warn("memcached related components are missing, functions may not work properly")
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			err := Resume(args[0], ResumeEngine, args[1:]...)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
	}
	resumeCmd.Flags().BoolVarP(&ResumeEngine, "resumeengine", "r", false, "resume batch engine support(optional)")

	var destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy the registered container",
		Long:  "destroy command is the basic command of lpmx, which is used for destroying the registered container via id",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			err := Destroy(args[0])
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.WithFields(logrus.Fields{
					"container id": args[0],
				}).Info("container is destroyed")
				return
			}
		},
	}

	var SetId string
	var SetType string
	var SetProg string
	var SetVal string
	var SetMode bool
	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "set environment variables for container",
		Long:  "set command is an advanced comand of lpmx, which is used for setting environment variables of running containers, you should clearly know what you want before using this command, it will reduce the performance heavily",
		Args:  cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			err := checkCompleteness()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
			err = CheckAndStartMemcache()
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			}
		},

		Run: func(cmd *cobra.Command, args []string) {
			err := Set(SetId, SetType, SetProg, SetVal, SetMode)
			if err != nil {
				LOGGER.Fatal(err.Error())
				return
			} else {
				LOGGER.WithFields(logrus.Fields{
					"container id": SetId,
				}).Info("container is set with new environment variables")
				return
			}
		},
	}
	setCmd.Flags().StringVarP(&SetId, "id", "i", "", "required(container id, you can get the id by command 'lpmx list')")
	setCmd.MarkFlagRequired("id")
	setCmd.Flags().StringVarP(&SetType, "type", "t", "", "required('add_map','remove_map','add_exec', 'remove_exec')")
	setCmd.MarkFlagRequired("type")
	setCmd.Flags().StringVarP(&SetProg, "name", "n", "", "required(should be the name of libc 'system calls wrapper' or mapped program path)")
	setCmd.MarkFlagRequired("name")
	setCmd.Flags().StringVarP(&SetVal, "value", "v", "", "required in add mode(value(file1:replace_file1;file2:repalce_file2;) or a mapped path) while optional in remove mode")
	setCmd.Flags().BoolVarP(&SetMode, "filecache", "c", false, "use FileCache rather than Memcached(optional)")

	var uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "uninstall lpmx completely",
		Long:  "entirely uninstall everything of lpmx",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Uninstall()
			if err != nil {
				LOGGER.Error(err.Error())
				return
			}
		},
	}

	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "update dependencies",
		Long:  "update necessary libraries of lpmx",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Update()
			if err != nil {
				LOGGER.Error(err.Error())
				return
			} else {
				LOGGER.Info("DONE")
			}
		},
	}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "show the version of LPMX",
		Long:  "LPMX version",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			LOGGER.Info(fmt.Sprintf("LPMX Version: %s", VERSION))
		},
	}

	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx rootless container",
	}
	rootCmd.AddCommand(initCmd, destroyCmd, listCmd, setCmd, resumeCmd, getCmd, dockerCmd, singularityCmd, exposeCmd, uninstallCmd, versionCmd, downloadCmd, updateCmd)
	rootCmd.Execute()
}
