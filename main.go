package main

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/container"
	. "github.com/jasonyangshadow/lpmx/error"
	"github.com/spf13/cobra"
	"os"
)

var mem *MemContainers

func main() {
	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx is an absolutely new package manager and rootless container",
		Long:  "lpmx could not only manage the local packages installed on users' systems but also create and run your package in specific rootless container, which could provide users flexible runtime environments",
	}

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "init lpmx itself",
		Long:  "init command is the basic command of lpmx, which is used for initializing lpmx system and fundamental structures",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			confs := []string{".", "/tmp/lpmx_root"}
			var err *Error
			mem, err = Init(confs)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}

	var CreateSource string
	var CreateName string
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "create container based on the location of folder and the given name",
		Long:  "create command is the basic command of lpmx, which is used for creating container structure and patching the elf headers inside the contianer",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			con, err := mem.CreateContainer(CreateSource, CreateName)
			if err == nil {
				fmt.Println(fmt.Sprintf("Container with id: %s is successfully created, start it with id", con.Id))
			} else {
				fmt.Println(err)
				os.Exit(1)
			}
		},
	}
	createCmd.Flags().StringVar(&CreateSource, "source", "", "conda environment dir(required)")
	createCmd.MarkFlagRequired("source")
	createCmd.Flags().StringVar(&CreateName, "name", "", "container name(required)")
	createCmd.MarkFlagRequired("name")

	var RunId string
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "run container using container id",
		Long:  "run command is the basic command of lpmx, which is used for running rootless runtime environment for specific packages",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			con, err := mem.RunContainer(RunId)
			if err == nil {
				fmt.Println(fmt.Sprintf("Container with id: %s starts successfully", con.Id))
			} else {
				fmt.Println(err)
				os.Exit(1)
			}

		},
	}

	runCmd.Flags().StringVar(&RunId, "id", "", "container id(required)")
	runCmd.MarkFlagRequired("id")

	var DestroyId string
	var destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy the running container using id",
		Long:  "destroy command is the basic command of lpmx, which is used for destroying the running container, only container id is required",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			mem.DestroyContainer(DestroyId)
		},
	}
	destroyCmd.Flags().StringVar(&DestroyId, "id", "", "container id(required)")
	destroyCmd.MarkFlagRequired("id")

	rootCmd.AddCommand(initCmd, createCmd, runCmd, destroyCmd)
	rootCmd.Execute()
}
