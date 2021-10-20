package main

import (
	"fmt"
	"os"
	"time"

	"github.com/liamg/lambo/pkg/gateway"
	"github.com/liamg/lambo/pkg/invoker"
	"github.com/spf13/cobra"
)

func main() {

	listenAddr := "127.0.0.1:3000"
	timeout := time.Second * 30
	var environmentVariables []string

	var rootCmd = &cobra.Command{
		Use:  "lambo [lambda-path]",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			invokerOptions := []invoker.Option{
				invoker.OptionWithDebugLogging(),
				invoker.OptionWithMaxDuration(timeout),
				invoker.OptionWithEnvVars(environmentVariables),
			}

			path := args[0]
			l := invoker.New(path, invokerOptions...)
			go l.Launch()
			defer l.Close()
			time.Sleep(time.Second)

			gwOptions := []gateway.Option{
				gateway.OptionWithDebugLogging(),
			}

			gw := gateway.New(l, gwOptions...)

			if err := gw.ListenAndServe(listenAddr); err != nil {
				fmt.Fprintf(os.Stderr, "Server error: %s\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringArrayVarP(&environmentVariables, "env-var", "e", environmentVariables, "Add environment variable to expose to the lambda")
	rootCmd.Flags().StringVarP(&listenAddr, "listen-addr", "l", listenAddr, "The server will listen for requests on this address and route them to your local lambda function.")
	rootCmd.Flags().DurationVarP(&timeout, "timeout", "t", timeout, "Maximum duration to allow a single invocation of the lambda to run for.")
	rootCmd.Execute()
}
