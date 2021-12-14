package invoker

import (
	"time"
)

type Option func(i *Invoker)

func OptionWithDebugLogging(debugEnabled bool) Option {
	return func(i *Invoker) {
		i.debug = debugEnabled
	}
}

func OptionWithMaxDuration(d time.Duration) Option {
	return func(i *Invoker) {
		i.deadline = d
	}
}

func OptionWithEnvVars(envVars []string) Option {
	return func(i *Invoker) {
		i.envVars = envVars
	}
}
