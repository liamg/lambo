package gateway

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

func convertHTTPRequest(r *http.Request) events.APIGatewayProxyRequest {
	var gw events.APIGatewayProxyRequest

	body, err := ioutil.ReadAll(r.Body)
	if err == nil {
		gw.Body = string(body)
	}

	gw.Headers = map[string]string{}
	for key, val := range r.Header {
		if len(val) == 0 {
			continue
		}
		gw.Headers[key] = val[0]
	}

	gw.HTTPMethod = r.Method
	gw.Path = r.URL.Path

	gw.QueryStringParameters = map[string]string{}
	gw.MultiValueQueryStringParameters = map[string][]string{}

	for key, val := range r.URL.Query() {
		if len(val) == 1 {
			gw.QueryStringParameters[key] = val[0]
			gw.MultiValueQueryStringParameters[key] = val
			continue
		}
		for i, value := range val {
			gw.QueryStringParameters[fmt.Sprintf("%s[%d]", key, i)] = value
		}
		gw.MultiValueQueryStringParameters[key] = val
	}

	return gw
}

func convertAPIGWResponse(r events.APIGatewayProxyResponse, w http.ResponseWriter) {
	w.WriteHeader(r.StatusCode)
	for key, val := range r.Headers {
		w.Header().Set(key, val)
	}

	if r.IsBase64Encoded {
		data, err := base64.StdEncoding.DecodeString(r.Body)
		if err == nil {
			w.Write(data)
			return
		}
	}

	w.Write([]byte(r.Body))
}
