package entry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"unicode"

	"github.com/aws/aws-lambda-go/events"
	"github.com/liamg/lambo/pkg/event"
)

func isAsciiPrintable(s []byte) bool {
	for _, r := range []rune(string(s)) {
		if r > unicode.MaxASCII || !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func convertHTTPRequest(r *http.Request) event.InvocationEvent {
	var gw events.APIGatewayProxyRequest

	var encoded bool

	if r.Body != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err == nil {
			if isAsciiPrintable(body) {
				gw.Body = string(body)
			} else {
				encoded = true
				gw.Body = base64.StdEncoding.EncodeToString(body)
			}
		}
	}

	gw.IsBase64Encoded = encoded

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

	return event.InvocationEvent{
		EventType: event.APIGatewayEventType,
		EventBody: gw,
	}
}

func convertAPIGWResponse(eventResponse event.InvocationEventResponse, w http.ResponseWriter) error {

	if eventResponse.ResponseType != event.APIGatewayResponseType {
		return fmt.Errorf("response is not the expected type")
	}

	r := eventResponse.ResponseBody.(events.APIGatewayProxyResponse)

	w.WriteHeader(r.StatusCode)
	for key, val := range r.Headers {
		w.Header().Set(key, val)
	}

	if r.StatusCode == http.StatusNoContent {
		return nil
	}

	if len(r.Body) == 0 {
		return nil
	}

	if r.IsBase64Encoded {
		data, err := base64.StdEncoding.DecodeString(r.Body)
		if err == nil {
			w.Write(data)
			return nil
		}
	}

	_, err := w.Write([]byte(r.Body))
	return err
}

func extractInvocationEvent(r *http.Request) (e event.InvocationEvent, err error) {
	if r.Body != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return e, err
		}
		if err := json.Unmarshal(body, &e); err != nil {
			return e, err
		}
	}
	return e, nil
}

