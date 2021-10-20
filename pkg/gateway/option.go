package gateway

type Option func(g *Gateway)

func OptionWithDebugLogging() Option {
	return func(g *Gateway) {
		g.debug = true
	}
}
