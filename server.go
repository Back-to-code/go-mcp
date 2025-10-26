package mcp

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/invopop/jsonschema"
)

type Server struct {
	name  string
	tools []internalToolT
}

type internalToolT struct {
	Name        string                             `json:"name"`
	Description string                             `json:"description"`
	InputSchema *jsonschema.Schema                 `json:"inputSchema"`
	Handler     func(json.RawMessage) (any, error) `json:"-"`
}

type Tool[T any] struct {
	// Required fields
	Name        string
	Description string
	Handler     func(arguments T) (any, error)

	// Not required
	args T
}

func NewServer(name string) *Server {
	return &Server{
		name:  name,
		tools: []internalToolT{},
	}
}

func AddToolToServer[T any](server *Server, tool Tool[T]) {
	err := TryAddToolToServer(server, tool)
	if err != nil {
		panic(err)
	}
}

func TryAddToolToServer[T any](server *Server, tool Tool[T]) error {
	tool.Name = strings.TrimSpace(tool.Name)
	if tool.Name == "" {
		return errors.New("tool has no name")
	}

	tool.Description = strings.TrimSpace(tool.Description)
	if tool.Description == "" {
		return errors.New("tool has no description")
	}

	inputSchema := (&jsonschema.Reflector{
		DoNotReference: true,
	}).Reflect(tool.args)

	if tool.Handler == nil {
		return errors.New("tool has no handler")
	}

	server.tools = append(server.tools, internalToolT{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: inputSchema,
		Handler: func(rawArguments json.RawMessage) (any, error) {
			var arguments T
			err := json.Unmarshal(rawArguments, &arguments)
			if err != nil {
				return nil, err
			}

			return tool.Handler(arguments)
		},
	})
	return nil
}
