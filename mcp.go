package mcp

import "encoding/json"

const (
	Jsonrpc         = "2.0"
	protocolVersion = "2025-06-18"
)

type Req struct {
	Jsonrpc string          `json:"jsonrpc"`
	Id      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Res struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

type M map[string]any

// https://modelcontextprotocol.io/specification/2025-06-18/server/tools#tool-result
type ToolResponse struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func newToolResponse(isError bool, text string) ToolResponse {
	return ToolResponse{
		Content: []ToolContent{{
			Type: "text",
			Text: text,
		}},
		IsError: isError,
	}
}
