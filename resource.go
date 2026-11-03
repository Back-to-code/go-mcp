package mcp

import (
	"context"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// https://modelcontextprotocol.io/specification/2025-06-18/server/resources#error-handling
const resourceNotFoundCode = -32002

// Resource is a text document a client can list and read. A Uri with parameters, like docs://profiles/{id}, makes it
// a template that reads every uri matching it, with the values of its parameters. A parameter matches one path
// segment, percent decoded.
type Resource struct {
	Uri         string
	Name        string
	Read        func(ctx context.Context, params map[string]string) (string, error)
	Title       string
	Description string
	MimeType    string
}

type internalResourceT struct {
	Resource
	pattern *regexp.Regexp
	params  []string
}

type listedResource struct {
	Uri         string `json:"uri,omitempty"`
	UriTemplate string `json:"uriTemplate,omitempty"`
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourceContents struct {
	Uri      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

var (
	uriParam      = regexp.MustCompile(`\{([^{}]*)\}`)
	uriParamName  = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	uriParamValue = `([^/?#]+)`
)

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
	for _, existing := range server.resources {
		if existing.Uri == resource.Uri {
			return errors.New("there already is a resource with uri " + resource.Uri)
		}
	}

	resource.Name = strings.TrimSpace(resource.Name)
	if resource.Name == "" {
		return errors.New("resource has no name")
	}

	if resource.Read == nil {
		return errors.New("resource has no read function")
	}

	pattern, params, err := compileUri(resource.Uri)
	if err != nil {
		return err
	}

	resource.Title = strings.TrimSpace(resource.Title)
	resource.Description = strings.TrimSpace(resource.Description)
	resource.MimeType = strings.TrimSpace(resource.MimeType)
	server.resources = append(server.resources, internalResourceT{Resource: resource, pattern: pattern, params: params})
	return nil
}

// compileUri supports the {name} parameters of RFC 6570, not its operators like {+name} or {?name}.
func compileUri(uri string) (*regexp.Regexp, []string, error) {
	matches := uriParam.FindAllStringSubmatchIndex(uri, -1)

	var pattern strings.Builder
	pattern.WriteString("^")
	params := []string{}
	end := 0
	for _, match := range matches {
		pattern.WriteString(regexp.QuoteMeta(uri[end:match[0]]))

		name := uri[match[2]:match[3]]
		if !uriParamName.MatchString(name) {
			return nil, nil, errors.New("uri " + uri + " has parameter {" + name + "}, only {name} parameters are supported")
		}
		if slices.Contains(params, name) {
			return nil, nil, errors.New("uri " + uri + " has parameter {" + name + "} more than once")
		}
		params = append(params, name)
		pattern.WriteString(uriParamValue)

		end = match[1]
	}
	pattern.WriteString(regexp.QuoteMeta(uri[end:]) + "$")

	if strings.ContainsAny(uriParam.ReplaceAllString(uri, ""), "{}") {
		return nil, nil, errors.New("uri " + uri + " has an unclosed parameter")
	}
	if len(params) == 0 {
		return nil, nil, nil
	}
	return regexp.MustCompile(pattern.String()), params, nil
}

func (s *Server) listResources(templates bool) []listedResource {
	listed := []listedResource{}
	for _, resource := range s.resources {
		if (resource.pattern != nil) != templates {
			continue
		}

		entry := listedResource{
			Name:        resource.Name,
			Title:       resource.Title,
			Description: resource.Description,
			MimeType:    resource.MimeType,
		}
		if templates {
			entry.UriTemplate = resource.Uri
		} else {
			entry.Uri = resource.Uri
		}
		listed = append(listed, entry)
	}
	return listed
}

// matchResource prefers a resource without parameters, then takes the first template that matches in the order they
// were added.
func (s *Server) matchResource(uri string) (*internalResourceT, map[string]string) {
	for idx, resource := range s.resources {
		if resource.pattern == nil && resource.Uri == uri {
			return &s.resources[idx], map[string]string{}
		}
	}

	for idx, resource := range s.resources {
		if resource.pattern == nil {
			continue
		}
		values := resource.pattern.FindStringSubmatch(uri)
		if values == nil {
			continue
		}

		params, ok := decodeParams(resource.params, values[1:])
		if ok {
			return &s.resources[idx], params
		}
	}

	return nil, nil
}

func decodeParams(names []string, values []string) (map[string]string, bool) {
	params := map[string]string{}
	for idx, name := range names {
		value, err := url.PathUnescape(values[idx])
		if err != nil {
			return nil, false
		}
		params[name] = value
	}
	return params, true
}

func (s *Server) readResource(uri string, ctx context.Context) (any, error) {
	resource, params := s.matchResource(uri)
	if resource == nil {
		return nil, &Error{Code: resourceNotFoundCode, Message: "Resource not found: " + uri}
	}

	text, err := resource.Read(ctx, params)
	if err != nil {
		return nil, err
	}

	return M{"contents": []ResourceContents{{Uri: uri, MimeType: resource.MimeType, Text: text}}}, nil
}
