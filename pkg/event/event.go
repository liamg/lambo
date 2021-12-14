package event

const (
	APIGatewayEventType    = "events.APIGatewayProxyRequest"
	APIGatewayResponseType = "events.APIGatewayProxyResponse"
	Other = ""
)

type InvocationEvent struct {
	EventType string      `json:"event_type"`
	EventBody interface{} `json:"event_body"`
}

type InvocationEventResponse struct {
	ResponseType string      `json:"response_type"`
	ResponseBody interface{} `json:"response_body"`
}
