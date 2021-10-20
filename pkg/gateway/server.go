package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/liamg/lambo/pkg/invoker"
)

type Gateway struct {
	invoker *invoker.Invoker
	debug   bool
}

func New(i *invoker.Invoker, options ...Option) *Gateway {
	g := &Gateway{
		invoker: i,
	}
	for _, opt := range options {
		opt(g)
	}
	return g
}

func (g *Gateway) ListenAndServe(addr string, options ...Option) error {
	g.log("Starting API gateway at http://%s...", addr)
	return http.ListenAndServe(addr, http.HandlerFunc(g.handler))
}

func (g *Gateway) log(format string, args ...interface{}) {
	if !g.debug {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [gateway] %s\n", time.Now().Format(time.StampNano), msg)
}

func (g *Gateway) handler(w http.ResponseWriter, r *http.Request) {

	g.log("Request received: %s", r.URL)
	gwreq := convertHTTPRequest(r)

	g.log("Forwarding request to lambda...")
	resp, err := g.invoker.Invoke(gwreq)
	if err != nil {
		g.log("Error sending request to lambda: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	g.log("Invocation succeeded, forwarding response to client.")
	convertAPIGWResponse(*resp, w)
}
