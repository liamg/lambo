package entry

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/liamg/lambo/pkg/invoker"
)

type Proxy struct {
	invoker *invoker.Invoker
	debug   bool
}

func newProxy(i *invoker.Invoker, options ...Option) *Proxy {
	g := &Proxy{
		invoker: i,
	}
	for _, opt := range options {
		opt(g)
	}
	return g
}

func (p *Proxy) ListenAndServe(addr string) error {
	p.log("Starting trigger event proxy at http://%s...", addr)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	p.log("Trigger event proxy is ready!")
	return http.Serve(listener, http.HandlerFunc(p.handler))
}

func (p *Proxy) SetDebugging(enabled bool) {
	p.debug = enabled
}

func (p *Proxy) log(format string, args ...interface{}) {
	if !p.debug {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [trigger-proxy] %s\n", time.Now().Format(time.StampNano), msg)
}

func (p *Proxy) handler(w http.ResponseWriter, r *http.Request) {

	p.log("Request received: %s", r.URL)
	event, err := extractInvocationEvent(r)
	if err != nil {
		panic(err)
	}

	p.log("Forwarding event to lambda...")
	resp, err := p.invoker.Invoke(event)
	if err != nil {
		p.log("Error sending event to lambda: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	p.log("Invocation succeeded, forwarding response to client.")
	p.log("response: %#v", resp)

}
