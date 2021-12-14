package main

import (
	"time"

	"github.com/liamg/lambo/pkg/entry"
	"github.com/liamg/lambo/pkg/invoker"
	"github.com/spf13/cobra"
)

func main() {

	listenAddr := "127.0.0.1:3000"
	timeout := time.Second * 30
	lambdaType := "gateway"
	debugEnabled := false
	var environmentVariables []string

	var rootCmd = &cobra.Command{
		Use:  "lambo [lambda-path]",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			invokerOptions := []invoker.Option{
				invoker.OptionWithDebugLogging(debugEnabled),
				invoker.OptionWithMaxDuration(timeout),
				invoker.OptionWithEnvVars(environmentVariables),
			}

			path := args[0]
			l := invoker.New(path, invokerOptions...)
			go func() { _ = l.Launch() }()
			defer func() { _ = l.Close() }()
			time.Sleep(time.Second)

			gw, err := entry.NewEntryPoint(lambdaType, l, entry.OptionWithDebugLogging(debugEnabled))
			if err != nil {
				return err
			}

			return gw.ListenAndServe(listenAddr)
		},
	}

	rootCmd.Flags().StringArrayVarP(&environmentVariables, "env-var", "e", environmentVariables, "Add environment variable to expose to the lambda")
	rootCmd.Flags().StringVarP(&listenAddr, "listen-addr", "l", listenAddr, "The server will listen for requests on this address and route them to your local lambda function.")
	rootCmd.Flags().StringVar(&lambdaType, "type", lambdaType, "The type of lambda choose from [gateway, triggered]")
	rootCmd.Flags().DurationVarP(&timeout, "timeout", "t", timeout, "Maximum duration to allow a single invocation of the lambda to run for.")
	rootCmd.Flags().BoolVarP(&debugEnabled, "debug", "d", debugEnabled, "Enable debug logging.")
	_ = rootCmd.Execute()
}
