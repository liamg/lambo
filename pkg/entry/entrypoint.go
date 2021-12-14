package entry

import (
	"fmt"

	"github.com/liamg/lambo/pkg/invoker"
)

type LambdaType string

const (
	LambdaTypeAPIGateway LambdaType = "gateway"
	LambdaTypeProxy      LambdaType = "proxy"
)

type LambdaEntryPoint interface {
	ListenAndServe(addr string) error
	SetDebugging(enabled bool)
}

func NewEntryPoint(lambdaType string, i *invoker.Invoker, options ...Option) (LambdaEntryPoint, error) {
	switch LambdaType(lambdaType) {
	case LambdaTypeAPIGateway:
		return newGateway(i, options...), nil
	case LambdaTypeProxy:
		return newProxy(i, options...), nil
	default:
		return nil, fmt.Errorf("lamdaType '%s' not recognised", lambdaType)
	}
}
