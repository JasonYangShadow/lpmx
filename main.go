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
			vals, err := List()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				fmt.Println(vals)
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

	var ResumeId string
	var resumeCmd = &cobra.Command{
		Use:   "resume",
		Short: "resume the registered container",
		Long:  "resume command is the basic command of lpmx, which is used for resuming the registered container via id",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Resume(ResumeId)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	resumeCmd.Flags().StringVarP(&ResumeId, "id", "i", "", "required")
	resumeCmd.MarkFlagRequired("id")

	var DestroyId string
	var destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy the registered container",
		Long:  "destroy command is the basic command of lpmx, which is used for destroying the registered container via id",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			err := Destroy(DestroyId)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			} else {
				fmt.Println(fmt.Sprintf("container: %s is destroyed", DestroyId))
			}
		},
	}
	destroyCmd.Flags().StringVarP(&DestroyId, "id", "i", "", "required")
	destroyCmd.MarkFlagRequired("id")

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
	setCmd.Flags().StringVarP(&SetId, "id", "i", "", "required")
	setCmd.MarkFlagRequired("id")
	setCmd.Flags().StringVarP(&SetType, "type", "t", "", "required")
	setCmd.MarkFlagRequired("type")
	setCmd.Flags().StringVarP(&SetProg, "name", "n", "", "required")
	setCmd.MarkFlagRequired("name")
	setCmd.Flags().StringVarP(&SetVal, "value", "v", "", "required")
	setCmd.MarkFlagRequired("value")

	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx rootless container",
	}
	rootCmd.AddCommand(runCmd)
	rootCmd.Execute()
}
