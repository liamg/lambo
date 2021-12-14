package entry

type Option func(g LambdaEntryPoint)

func OptionWithDebugLogging(debugEnabled bool) Option {

	return func(g LambdaEntryPoint) {
		g.SetDebugging(debugEnabled)
	}

}
