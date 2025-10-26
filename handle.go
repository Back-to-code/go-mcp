package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

func (s *Server) Handle(method string, body []byte) Response {
	switch method {
	case "GET":
		return Response{
			Status:      200,
			ContentType: "plain/text",
			Payload:     []byte("Ok"),
		}
	case "POST", "PATCH", "PUT":
		return s.handleStreamableHttpRequest(body)
	default:
		return Response{
			Status:      404,
			ContentType: "plain/text",
			Payload:     []byte("404 Not found"),
		}
	}
}

func (s *Server) handleStreamableHttpRequest(body []byte) Response {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return emptyResponse
	}

	requests := []Req{}
	var err error
	if body[0] == '{' {
		// This is a single request
		var req Req
		err = json.Unmarshal(body, &req)
		requests = []Req{req}
	} else {
		// This is a batch request
		err = json.Unmarshal(body, &requests)
	}
	if err != nil {
		return errorResponse(err)
	}

	resp := []any{}
outer:
	for _, req := range requests {
		addResp := func(result any) Res {
			res := Res{
				Jsonrpc: Jsonrpc,
				Id:      req.Id,
				Result:  result,
			}
			resp = append(resp, res)
			return res
		}
		addErr := func(err error) Res {
			res := Res{
				Jsonrpc: Jsonrpc,
				Id:      req.Id,
				Error: &Error{
					Message: err.Error(),
				},
			}
			resp = append(resp, res)
			return res
		}

		switch req.Method {
		case "initialize":
			return jsonResponse(addResp(M{
				"protocolVersion": protocolVersion,
				"capabilities": M{
					"tools": M{
						// We do not actually support this, but this might be required by claude??
						"listChanged": true,
					},
				},
				"serverInfo": M{
					"name":    "test mcp",
					"version": "1.0.0",
				},
			}))
		case "ping":
			addResp(M{})
		case "notifications/initialized":
			// FIXME
			resp = append(resp, Req{
				Jsonrpc: "2.0",
				Method:  "notifications/tools/list_changed",
			})
		case "notifications/cancelled":
			// FIXME: https://modelcontextprotocol.io/specification/2025-03-26/basic/utilities/cancellation
		case "tools/list":
			if s.tools == nil {
				s.tools = []internalToolT{}
			}

			addResp(M{
				"tools": s.tools,
			})
		case "tools/call":
			var params struct {
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
			}
			err := json.Unmarshal(req.Params, &params)
			if err != nil {
				addErr(err)
				continue
			}

			var pickedTool *internalToolT
			for _, tool := range s.tools {
				if tool.Name == params.Name {
					pickedTool = &tool
					break
				}
			}
			if pickedTool == nil {
				addErr(errors.New("no tool named " + params.Name))
				continue
			}

			out, err := pickedTool.Handler(params.Arguments)
			if err != nil {
				addResp(newToolResponse(true, err.Error()))
				continue
			}

			outReflection := reflect.ValueOf(out)
			for outReflection.Kind() == reflect.Ptr {
				if outReflection.IsNil() {
					addResp(newToolResponse(false, "null"))
					continue outer
				}
				outReflection = outReflection.Elem()
			}

			switch outReflection.Kind() {
			case reflect.String:
				addResp(newToolResponse(false, outReflection.String()))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
				addResp(newToolResponse(false, fmt.Sprintf("%v", out)))
			case reflect.Chan, reflect.Func, reflect.Pointer, reflect.Uintptr:
				addResp(newToolResponse(true, "returned value that cannot be converted to a string"))
			default:
				outJson, err := json.Marshal(out)
				if err == nil {
					addResp(newToolResponse(true, err.Error()))
				} else {
					addResp(newToolResponse(false, string(outJson)))
				}
			}
		case "prompts/list":
			addResp(M{
				"prompts": []M{},
			})
		case "resources/list":
			addResp(M{
				"resources": []M{},
			})
		}
	}

	var toMarshall any = resp
	switch len(resp) {
	case 0:
		return emptyResponse
	case 1:
		toMarshall = resp[0]
	}

	return jsonResponse(toMarshall)
}
