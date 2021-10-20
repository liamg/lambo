package invoker

import "time"

type Option func(i *Invoker)

func OptionWithDebugLogging() Option {
	return func(i *Invoker) {
		i.debug = true
	}
}

func OptionWithMaxDuration(d time.Duration) Option {
	return func(i *Invoker) {
		i.deadline = d
	}
}
