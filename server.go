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

func (t Tool[T]) AddToServer(s *Server) {
	err := t.TryAddToServer(s)
	if err != nil {
		panic(err)
	}
}

func (t Tool[T]) TryAddToServer(s *Server) error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return errors.New("tool has no name")
	}

	t.Description = strings.TrimSpace(t.Description)
	if t.Description == "" {
		return errors.New("tool has no description")
	}

	inputSchema := (&jsonschema.Reflector{
		DoNotReference: true,
	}).Reflect(t.args)

	if t.Handler == nil {
		return errors.New("tool has no handler")
	}

	s.tools = append(s.tools, internalToolT{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: inputSchema,
		Handler: func(rawArguments json.RawMessage) (any, error) {
			var arguments T
			err := json.Unmarshal(rawArguments, &arguments)
			if err != nil {
				return nil, err
			}

			return t.Handler(arguments)
		},
	})
	return nil
}
