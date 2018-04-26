package main

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/container"
	"github.com/spf13/cobra"
	"os"
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
				fmt.Println(err)
				os.Exit(1)
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
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	var RunSource string
	var RunConfig string
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "run container based on specific directory",
		Long:  "run command is the basic command of lpmx, which is used for initializing, creating and running container based on specific directory",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Run(RunSource, RunConfig)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	runCmd.Flags().StringVarP(&RunSource, "source", "s", "", "required")
	runCmd.MarkFlagRequired("source")
	runCmd.Flags().StringVarP(&RunConfig, "config", "c", "", "required")
	runCmd.MarkFlagRequired("config")

	var resumeCmd = &cobra.Command{
		Use:   "resume",
		Short: "resume the registered container",
		Long:  "resume command is the basic command of lpmx, which is used for resuming the registered container via id",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			err := Resume(args[0])
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
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
				fmt.Println(err)
				os.Exit(1)
			} else {
				fmt.Println(fmt.Sprintf("container: %s is destroyed", args[0]))
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
				fmt.Println(err)
				os.Exit(1)
			} else {
				fmt.Println(fmt.Sprintf("container %s is set with new environment variables", SetId))
			}
		},
	}
	setCmd.Flags().StringVarP(&SetId, "id", "i", "", "required(container id, you can get the id by command 'lpmx list')")
	setCmd.MarkFlagRequired("id")
	setCmd.Flags().StringVarP(&SetType, "type", "t", "", "required('add_needed', 'remove_needed', 'add_rpath', 'remove_rpath', 'change_user', 'add_privilege', 'remove_privilege')")
	setCmd.MarkFlagRequired("type")
	setCmd.Flags().StringVarP(&SetProg, "name", "n", "", "required(put 'user' for operation change_user)")
	setCmd.MarkFlagRequired("name")
	setCmd.Flags().StringVarP(&SetVal, "value", "v", "", "value (optional for removing privilege operation)")

	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx rootless container",
	}
	rootCmd.AddCommand(initCmd, destroyCmd, listCmd, runCmd, setCmd, resumeCmd)
	rootCmd.Execute()
}
