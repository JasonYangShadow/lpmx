package main

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/container"
	"github.com/spf13/cobra"
	"os"
)

func main() {
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

	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx rootless container",
	}
	rootCmd.AddCommand(runCmd)
	rootCmd.Execute()
}
