package invoker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/liamg/lambo/pkg/event"
)

const (
	envRPCPort    = "_LAMBDA_SERVER_PORT"
	envServerAddr = "AWS_LAMBDA_RUNTIME_API"

	headerAWSRequestID       = "Lambda-Runtime-Aws-Request-Id"
	headerDeadlineMS         = "Lambda-Runtime-Deadline-Ms"
	headerTraceID            = "Lambda-Runtime-Trace-Id"
	headerCognitoIdentity    = "Lambda-Runtime-Cognito-Identity"
	headerClientContext      = "Lambda-Runtime-Client-Context"
	headerInvokedFunctionARN = "Lambda-Runtime-Invoked-Function-Arn"

	contentTypeJSON   = "application/json"
	defaultAPIVersion = "2018-06-01"
)

type Invoker struct {
	lambdaPath     string
	dir            string
	args           []string
	apiVersion     string
	invocationChan chan Invocation
	listener       net.Listener
	deadline       time.Duration
	invMu          sync.Mutex
	invocations    map[string]Invocation
	debug          bool
	envVars        []string
}

type Invocation struct {
	ID       string
	Request  interface{}
	respChan chan InvocationResult
}

type InvocationResult struct {
	Response interface{}
	Error    error
}

func New(lambdaPath string, options ...Option) *Invoker {
	l := &Invoker{
		lambdaPath:     lambdaPath,
		apiVersion:     defaultAPIVersion,
		invocationChan: make(chan Invocation, 0x10),
		deadline:       time.Second * 10,
		invocations:    make(map[string]Invocation),
	}
	for _, opt := range options {
		opt(l)
	}
	return l
}

func (i *Invoker) Launch() error {

	absolutePath, err := filepath.Abs(i.lambdaPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path to lambda: %w", err)
	}
	workingDir := filepath.Dir(absolutePath)
	if i.dir != "" {
		workingDir = i.dir
	}

	i.log("Lambda function is at %s", absolutePath)
	i.log("Working directory set to %s", workingDir)

	stat, err := os.Stat(absolutePath)
	if err != nil {
		return fmt.Errorf("failed to stat lambda: %w", err)
	}

	if stat.IsDir() {
		return fmt.Errorf("lambda is a directory")
	}

	if stat.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("lambda is not executable")
	}

	listenAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	i.listener, err = net.ListenTCP("tcp", listenAddr)
	if err != nil {
		return err
	}

	i.log("Started listener at %s", i.listener.Addr().String())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := append(i.envVars, []string{
		fmt.Sprintf("%s=%s", envServerAddr, i.listener.Addr().String()), // disable rpc
		fmt.Sprintf("%s=%s", envRPCPort, ""),                            // disable rpc
	}...)

	cmd := exec.CommandContext(ctx, absolutePath, i.args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	i.log("Launching lambda...")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch lambda: %w", err)
	}

	basePattern := fmt.Sprintf("/%s/runtime/invocation/", i.apiVersion)

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("%snext", basePattern), i.handleNext)
	mux.HandleFunc("/", i.handleResult)

	i.log("Starting lambda API server...")
	if err := http.Serve(i.listener, mux); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (i *Invoker) log(format string, args ...interface{}) {
	if !i.debug {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [invoker] %s\n", time.Now().Format(time.StampNano), msg)
}

func (i *Invoker) Close() error {
	i.log("Stopping...")
	return i.listener.Close()
}

func (i *Invoker) Invoke(e event.InvocationEvent) (*event.InvocationEventResponse, error) {
	switch e.EventType {
	case event.APIGatewayEventType:
		request := e.EventBody.(events.APIGatewayProxyRequest)
		response, err := i.invokeAPIGateway(request)
		if err != nil {
			return nil, err
		}
		return response, nil
	default:
		request := e.EventBody
		response, err := i.invokeEvent(request)
		if err != nil {
			return nil, err
		}
		return response, nil
	}
}

func (i *Invoker) invokeEvent(request interface{}) (*event.InvocationEventResponse, error) {

	i.log("Invoking with a gateway request")
	respChan := make(chan InvocationResult)

	invocation := Invocation{
		Request:  request,
		ID:       uuid.New().String(),
		respChan: respChan,
	}

	i.invMu.Lock()
	i.invocations[invocation.ID] = invocation
	i.invMu.Unlock()

	defer func() {
		i.invMu.Lock()
		delete(i.invocations, invocation.ID)
		i.invMu.Unlock()
	}()

	i.invocationChan <- invocation
	result := <-invocation.respChan

	if result.Error != nil {
		i.log("Invocation %s failed: %s", invocation.ID, result.Error)
		return nil, result.Error
	}

	response := result.Response
	return &event.InvocationEventResponse{
		ResponseType: event.Other,
		ResponseBody: response,
	}, nil
}

func (i *Invoker) invokeAPIGateway(request events.APIGatewayProxyRequest) (*event.InvocationEventResponse, error) {

	i.log("Invoking with a gateway request")
	respChan := make(chan InvocationResult)

	invocation := Invocation{
		Request:  request,
		ID:       uuid.New().String(),
		respChan: respChan,
	}

	i.invMu.Lock()
	i.invocations[invocation.ID] = invocation
	i.invMu.Unlock()

	defer func() {
		i.invMu.Lock()
		delete(i.invocations, invocation.ID)
		i.invMu.Unlock()
	}()

	i.log("Invoking lambda for %s:%s, with url path '%s'...", invocation.ID, request.HTTPMethod, request.Path)
	i.invocationChan <- invocation
	result := <-invocation.respChan

	if result.Error != nil {
		i.log("Invocation %s failed: %s", invocation.ID, result.Error)
		return nil, result.Error
	}

	response := result.Response.(events.APIGatewayProxyResponse)
	i.log("Invocation %s succeeded, status code %d, body length %d.", invocation.ID, response.StatusCode, len(response.Body))
	return &event.InvocationEventResponse{
		ResponseType: event.APIGatewayResponseType,
		ResponseBody: response,
	}, nil
}

func (i *Invoker) handleNext(w http.ResponseWriter, r *http.Request) {

	i.log("Lambda client connected, waiting for requests...")

	invocation := <-i.invocationChan

	// set request id
	w.Header().Set(headerAWSRequestID, invocation.ID)

	// set deadline
	w.Header().Set(headerDeadlineMS, fmt.Sprintf("%d", i.deadline.Milliseconds()))

	// todo: set some more headers?

	body, err := json.Marshal(invocation.Request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		i.sendInvocationResult(invocation.ID, InvocationResult{
			Error: fmt.Errorf("failed to send job: %s", err),
		})
		return
	}

	i.log("Sending invocation job for %s to client...", invocation.ID)
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		return
	}
}

func (i *Invoker) handleResult(w http.ResponseWriter, r *http.Request) {

	if !strings.Contains(r.URL.Path, "/invocation/") {
		w.WriteHeader(http.StatusNotFound)
		i.log("Unexpected request for %s", r.URL)
		return
	}

	parts := strings.SplitN(r.URL.Path, "/invocation/", 2)
	segments := strings.Split(parts[1], "/")
	if len(segments) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		i.log("Bad request for %s", r.URL)
		return
	}

	id := segments[0]
	switch segments[1] {
	case "error":
		i.handleError(id, w, r)
	case "response":
		i.handleResponse(id, w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func (i *Invoker) handleError(id string, w http.ResponseWriter, r *http.Request) {

	i.log("Received error response for invocation %s", id)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		i.sendInvocationResult(id, InvocationResult{
			Error: fmt.Errorf("failed to handle error: %s", err),
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	i.sendInvocationResult(id, InvocationResult{
		Error: fmt.Errorf("%s", string(body)),
	})
}

func (i *Invoker) handleResponse(id string, w http.ResponseWriter, r *http.Request) {

	i.log("Received success response for invocation %s", id)

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		i.sendInvocationResult(id, InvocationResult{
			Error: fmt.Errorf("failed to handle response: %s", err),
		})
		return
	}

	var respBody events.APIGatewayProxyResponse
	if err := json.Unmarshal(body, &respBody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		i.sendInvocationResult(id, InvocationResult{
			Error: fmt.Errorf("failed to unmarshal response: %s", err),
		})
		return
	}

	w.WriteHeader(http.StatusAccepted)
	i.sendInvocationResult(id, InvocationResult{
		Response: respBody,
	})
}

func (i *Invoker) sendInvocationResult(id string, result InvocationResult) {
	i.invMu.Lock()
	invocation, ok := i.invocations[id]
	i.invMu.Unlock()
	if !ok {
		// invocation not found - this should not happen...
		return
	}

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case invocation.respChan <- result:
	case <-timer.C:
	}
}
