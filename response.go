package mcp

import "encoding/json"

type Response struct {
	Status      int
	ContentType string
	Payload     []byte // might be nil!
}

func (r Response) Ok() bool {
	return r.Status == 200
}

var emptyResponse = Response{
	Status:      201,
	ContentType: "application/json",
}

func jsonResponse(res any) Response {
	payloadJson, err := json.Marshal(res)
	if err != nil {
		return errorResponse(err)
	}

	return Response{
		Status:      200,
		ContentType: "application/json",
		Payload:     payloadJson,
	}
}

func errorResponse(err error) Response {
	payloadJson, err := json.Marshal(Res{
		Jsonrpc: Jsonrpc,
		Error: &Error{
			Message: err.Error(),
		},
	})
	if err != nil {
		// Fallback to the simple error response as if we run error response again we might get a circular error
		return simpleErrorResponse(err)
	}

	return Response{
		Status:      400,
		ContentType: "application/json",
		Payload:     payloadJson,
	}
}

func simpleErrorResponse(err error) Response {
	return Response{
		Status:      400,
		ContentType: "plain/text",
		Payload:     []byte(err.Error()),
	}
}
