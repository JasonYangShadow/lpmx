package main

import (
	. "github.com/jasonyangshadow/lpmx/container"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx is an absolutely new package manager and rootless container",
		Long:  "lpmx could not only manage the local packages installed on users' systems but also create and run your package in specific rootless container, which could provide users flexible runtime environments",
	}

	var CreateSource string
	var CreateName string
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "create container based on the location of folder and the given name",
		Long:  "create command is the basic command of lpmx, which is used for creating container structure and patching the elf headers inside the contianer",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {

		},
	}

	var RunSource string
	var RunName string

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "create and run container based on the locaiton of conda environment",
		Long:  "run command is the basic command of lpmx, which is used for creating and running rootless runtime environment for specific packages",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			RunContainer(args[0], args[1])
		},
	}

	runCmd.Flags().StringVar(&RunSource, "source", "", "conda environment dir(required)")
	runCmd.MarkFlagRequired("source")
	runCmd.Flags().StringVar(&RunName, "name", "", "container name(required)")
	runCmd.MarkFlagRequired("name")

	var DestroyName string

	var destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "destroy the running container",
		Long:  "destroy command is the basic command of lpmx, which is used for destroying the running container, only container name is required",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			DestroyContainer(args[0])
		},
	}

	destroyCmd.Flags().StringVar(&DestroyName, "name", "", "container name(required)")
	destroyCmd.MarkFlagRequired("name")

	rootCmd.AddCommand(runCmd, destroyCmd)
	rootCmd.Execute()
}
