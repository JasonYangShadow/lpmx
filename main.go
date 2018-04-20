package main

import (
	"fmt"
	. "github.com/jasonyangshadow/lpmx/container"
	"github.com/spf13/cobra"
	"os"
)

func run(dir string, name string) {
	confs := []string{"/tmp/lpmx_root"}
	mem, ierr := Init(confs)
	if ierr == nil {
		con, cerr := mem.CreateContainer(dir, name)
		if cerr == nil {
			_, rerr := mem.RunContainer(con.Id)
			if rerr != nil {
				fmt.Println(rerr)
				os.Exit(1)
			}
		} else {
			fmt.Println(cerr)
			os.Exit(1)
		}
	} else {
		fmt.Println(ierr)
		os.Exit(1)
	}
}

func main() {
	var RunSource string
	var RunName string
	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "run container based on specific directory",
		Long:  "run command is the basic command of lpmx, which is used for initializing, creating and running container based on specific directory",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			run(RunSource, RunName)
		},
	}
	runCmd.Flags().StringVarP(&RunSource, "source", "s", "", "required")
	runCmd.MarkFlagRequired("source")
	runCmd.Flags().StringVarP(&RunName, "name", "n", "", "required")
	runCmd.MarkFlagRequired("name")

	var rootCmd = &cobra.Command{
		Use:   "lpmx",
		Short: "lpmx rootless container",
	}
	rootCmd.AddCommand(runCmd)
	rootCmd.Execute()
}
