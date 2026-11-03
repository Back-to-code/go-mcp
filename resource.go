package mcp

import (
	"context"
	"errors"
	"strings"
)

// https://modelcontextprotocol.io/specification/2025-06-18/server/resources#error-handling
const resourceNotFoundCode = -32002

// Resource is a text document a client can list and read.
type Resource struct {
	// Required fields
	Uri  string
	Name string
	Read func(ctx context.Context) (string, error)

	// Not required
	Title       string
	Description string
	MimeType    string
}

type internalResourceT struct {
	Uri         string                                    `json:"uri"`
	Name        string                                    `json:"name"`
	Title       string                                    `json:"title,omitempty"`
	Description string                                    `json:"description,omitempty"`
	MimeType    string                                    `json:"mimeType,omitempty"`
	Read        func(ctx context.Context) (string, error) `json:"-"`
}

type ResourceContents struct {
	Uri      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

func AddResourceToServer(server *Server, resource Resource) {
	err := TryAddResourceToServer(server, resource)
	if err != nil {
		panic(err)
	}
}

func TryAddResourceToServer(server *Server, resource Resource) error {
	resource.Uri = strings.TrimSpace(resource.Uri)
	if resource.Uri == "" {
		return errors.New("resource has no uri")
	}
	if server.resource(resource.Uri) != nil {
		return errors.New("there already is a resource with uri " + resource.Uri)
	}

	resource.Name = strings.TrimSpace(resource.Name)
	if resource.Name == "" {
		return errors.New("resource has no name")
	}

	if resource.Read == nil {
		return errors.New("resource has no read function")
	}

	server.resources = append(server.resources, internalResourceT{
		Uri:         resource.Uri,
		Name:        resource.Name,
		Title:       strings.TrimSpace(resource.Title),
		Description: strings.TrimSpace(resource.Description),
		MimeType:    strings.TrimSpace(resource.MimeType),
		Read:        resource.Read,
	})
	return nil
}

func (s *Server) resource(uri string) *internalResourceT {
	for idx := range s.resources {
		if s.resources[idx].Uri == uri {
			return &s.resources[idx]
		}
	}
	return nil
}

func (s *Server) readResource(uri string, ctx context.Context) (any, error) {
	resource := s.resource(uri)
	if resource == nil {
		return nil, &Error{Code: resourceNotFoundCode, Message: "Resource not found: " + uri}
	}

	text, err := resource.Read(ctx)
	if err != nil {
		return nil, err
	}

	return M{"contents": []ResourceContents{{Uri: resource.Uri, MimeType: resource.MimeType, Text: text}}}, nil
}
