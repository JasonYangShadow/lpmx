package main

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/container"
	. "github.com/jasonyangshadow/lpmx/log"
	. "github.com/jasonyangshadow/lpmx/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"path/filepath"
	"strconv"
)

func main() {

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "init the lpmx itself",
		Long:  "init command is the basic command of lpmx, which is used for initializing lpmx system",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Init()
			if err != nil {
				LOGGER.Panic(err.Error())
			}
		},
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "list the containers in lpmx system",
		Long:  "list command is the basic command of lpmx, which is used for listing all the containers registered",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := List()
			if err != nil {
				LOGGER.Panic(err.Error())
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
		Run: func(cmd *cobra.Command, args []string) {
			RunSource, _ = filepath.Abs(RunSource)
			if RunConfig != "" {
				RunConfig, _ = filepath.Abs(RunConfig)
			} else {
				config_path := fmt.Sprintf("%s/setting.yml", RunSource)
				if FileExist(config_path) {
					RunConfig = config_path
				} else {
					LOGGER.Panic("can't locate the setting.yml in source folder")
				}
			}
			err := Run(RunSource, RunConfig, RunPassive)
			if err != nil {
				LOGGER.Panic(err.Error())
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
		Run: func(cmd *cobra.Command, args []string) {
			err := Get(GetId, GetName)
			if err != nil {
				LOGGER.Panic(err.Error())
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
		Run: func(cmd *cobra.Command, args []string) {
			_, err := RPCExec(RExecIp, RExecPort, RExecTimeout, args[0], args[1:]...)
			if err != nil {
				LOGGER.Panic(err.Error())
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
		Run: func(cmd *cobra.Command, args []string) {
			res, err := RPCQuery(RQueryIp, RQueryPort)
			if err != nil {
				LOGGER.Panic(err.Error())
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
		Run: func(cmd *cobra.Command, args []string) {
			i, aerr := strconv.Atoi(RDeletePid)
			if aerr != nil {
				LOGGER.Panic(aerr.Error())
			}
			_, err := RPCDelete(RDeleteIp, RDeletePort, i)
			if err != nil {
				LOGGER.Panic(err.Error())
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

	var resumeCmd = &cobra.Command{
		Use:   "resume",
		Short: "resume the registered container",
		Long:  "resume command is the basic command of lpmx, which is used for resuming the registered container via id",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := Resume(args[0])
			if err != nil {
				LOGGER.Panic(err.Error())
			}
		},
	}

	var destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy the registered container",
		Long:  "destroy command is the basic command of lpmx, which is used for destroying the registered container via id",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := Destroy(args[0])
			if err != nil {
				LOGGER.Panic(err.Error())
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
		Long:  "set command is an additional comand of lpmx, which is used for setting environment variables of running containers",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Set(SetId, SetType, SetProg, SetVal)
			if err != nil {
				LOGGER.Panic(err.Error())
			} else {
				LOGGER.WithFields(logrus.Fields{
					"container id": SetId,
				}).Info("container is set with new environment variables")
			}
		},
	}
	setCmd.Flags().StringVarP(&SetId, "id", "i", "", "required(container id, you can get the id by command 'lpmx list')")
	setCmd.MarkFlagRequired("id")
	setCmd.Flags().StringVarP(&SetType, "type", "t", "", "required('add_needed', 'remove_needed', 'add_rpath', 'remove_rpath', 'change_user', 'add_allow_priv', 'remove_allow_priv','add_deny_priv','remove_deny_priv','add_map','remove_map')")
	setCmd.MarkFlagRequired("type")
	setCmd.Flags().StringVarP(&SetProg, "name", "n", "", "required(put 'user' for operation change_user)")
	setCmd.MarkFlagRequired("name")
	setCmd.Flags().StringVarP(&SetVal, "value", "v", "", "value (optional for removing privilege operation or removing map operation)")

	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx rootless container",
	}
	rootCmd.AddCommand(initCmd, destroyCmd, listCmd, runCmd, setCmd, resumeCmd, rpcCmd, getCmd)
	rootCmd.Execute()
}
