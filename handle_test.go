package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type initializeResult struct {
	Result struct {
		Instructions *string `json:"instructions"`
		ServerInfo   struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
	} `json:"result"`
}

func initialize(t *testing.T, server *Server) initializeResult {
	t.Helper()

	res := server.Handle("POST", []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`), context.Background())
	if !res.Ok() {
		t.Fatalf("initialize failed with status %d: %s", res.Status, res.Payload)
	}

	var result initializeResult
	err := json.Unmarshal(res.Payload, &result)
	if err != nil {
		t.Fatalf("initialize returned invalid json: %v", err)
	}

	return result
}

func TestInitializeReturnsTheInstructions(t *testing.T) {
	server := NewServer("Example server")
	server.SetInstructions("  Read the guide first  ")

	result := initialize(t, server)

	if result.Result.Instructions == nil || *result.Result.Instructions != "Read the guide first" {
		t.Fatalf("expected the trimmed instructions, got %v", result.Result.Instructions)
	}
	if result.Result.ServerInfo.Name != "Example server" {
		t.Fatalf("expected the server name, got %q", result.Result.ServerInfo.Name)
	}
}

func TestInitializeLeavesOutEmptyInstructions(t *testing.T) {
	result := initialize(t, NewServer("Example server"))

	if result.Result.Instructions != nil {
		t.Fatalf("expected no instructions, got %q", *result.Result.Instructions)
	}
}

func request(t *testing.T, server *Server, body string, result any) {
	t.Helper()

	res := server.Handle("POST", []byte(body), context.Background())
	if !res.Ok() {
		t.Fatalf("request failed with status %d: %s", res.Status, res.Payload)
	}

	err := json.Unmarshal(res.Payload, result)
	if err != nil {
		t.Fatalf("request returned invalid json: %v", err)
	}
}

func resourceServer(t *testing.T) *Server {
	t.Helper()

	server := NewServer("Example server")
	AddResourceToServer(server, Resource{
		Uri:         "docs://guide",
		Name:        "guide",
		Title:       "Guide",
		Description: "How to use the tools",
		MimeType:    "text/markdown",
		Read: func(_ context.Context, _ map[string]string) (string, error) {
			return "# Guide", nil
		},
	})
	return server
}

func TestInitializeDeclaresResourcesOnlyWhenThereAreSome(t *testing.T) {
	var capabilities struct {
		Result struct {
			Capabilities map[string]any `json:"capabilities"`
		} `json:"result"`
	}
	initializeBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`

	request(t, NewServer("Example server"), initializeBody, &capabilities)
	if _, ok := capabilities.Result.Capabilities["resources"]; ok {
		t.Fatal("expected no resources capability without resources")
	}

	request(t, resourceServer(t), initializeBody, &capabilities)
	if _, ok := capabilities.Result.Capabilities["resources"]; !ok {
		t.Fatal("expected the resources capability")
	}
}

func TestResourcesList(t *testing.T) {
	var list struct {
		Result struct {
			Resources []map[string]any `json:"resources"`
		} `json:"result"`
	}
	request(t, resourceServer(t), `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`, &list)

	if len(list.Result.Resources) != 1 {
		t.Fatalf("expected one resource, got %v", list.Result.Resources)
	}
	resource := list.Result.Resources[0]
	expected := map[string]any{
		"uri":         "docs://guide",
		"name":        "guide",
		"title":       "Guide",
		"description": "How to use the tools",
		"mimeType":    "text/markdown",
	}
	for key, value := range expected {
		if resource[key] != value {
			t.Fatalf("expected %s to be %v, got %v", key, value, resource[key])
		}
	}
}

func TestResourcesRead(t *testing.T) {
	var read struct {
		Result struct {
			Contents []ResourceContents `json:"contents"`
		} `json:"result"`
	}
	request(t, resourceServer(t), `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"docs://guide"}}`, &read)

	expected := []ResourceContents{{Uri: "docs://guide", MimeType: "text/markdown", Text: "# Guide"}}
	if len(read.Result.Contents) != 1 || read.Result.Contents[0] != expected[0] {
		t.Fatalf("expected %v, got %v", expected, read.Result.Contents)
	}
}

func TestResourcesReadOfAnUnknownUri(t *testing.T) {
	var read struct {
		Error *Error `json:"error"`
	}
	request(t, resourceServer(t), `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"docs://unknown"}}`, &read)

	if read.Error == nil || read.Error.Code != resourceNotFoundCode {
		t.Fatalf("expected a resource not found error, got %v", read.Error)
	}
}

func TestResourcesReadPassesOnTheReadError(t *testing.T) {
	server := NewServer("Example server")
	AddResourceToServer(server, Resource{
		Uri:  "docs://broken",
		Name: "broken",
		Read: func(_ context.Context, _ map[string]string) (string, error) {
			return "", errors.New("could not read")
		},
	})

	var read struct {
		Error *Error `json:"error"`
	}
	request(t, server, `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"docs://broken"}}`, &read)

	if read.Error == nil || read.Error.Message != "could not read" {
		t.Fatalf("expected the read error, got %v", read.Error)
	}
}

func TestTryAddResourceToServerRejectsInvalidResources(t *testing.T) {
	read := func(_ context.Context, _ map[string]string) (string, error) { return "", nil }

	cases := map[string]Resource{
		"no uri":                  {Name: "guide", Read: read},
		"no name":                 {Uri: "docs://other", Read: read},
		"no read":                 {Uri: "docs://other", Name: "other"},
		"uri is taken":            {Uri: "docs://guide", Name: "guide", Read: read},
		"parameter operator":      {Uri: "docs://profiles/{+id}", Name: "profile", Read: read},
		"parameter twice":         {Uri: "docs://profiles/{id}/{id}", Name: "profile", Read: read},
		"unclosed parameter":      {Uri: "docs://profiles/{id", Name: "profile", Read: read},
		"parameter without close": {Uri: "docs://profiles/id}", Name: "profile", Read: read},
	}
	for name, resource := range cases {
		t.Run(name, func(t *testing.T) {
			err := TryAddResourceToServer(resourceServer(t), resource)
			if err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func templateServer(t *testing.T) *Server {
	t.Helper()

	server := resourceServer(t)
	AddResourceToServer(server, Resource{
		Uri:         "docs://profiles/{id}/rules/{field}",
		Name:        "profile-rules",
		Description: "The rules of one field of a profile",
		Read: func(_ context.Context, params map[string]string) (string, error) {
			return params["id"] + " " + params["field"], nil
		},
	})
	AddResourceToServer(server, Resource{
		Uri:  "docs://profiles/latest/rules/{field}",
		Name: "latest-profile-rules",
		Read: func(_ context.Context, params map[string]string) (string, error) {
			return "latest " + params["field"], nil
		},
	})
	return server
}

func readText(t *testing.T, server *Server, uri string) (string, *Error) {
	t.Helper()

	var read struct {
		Result struct {
			Contents []ResourceContents `json:"contents"`
		} `json:"result"`
		Error *Error `json:"error"`
	}
	body, err := json.Marshal(M{"jsonrpc": "2.0", "id": 1, "method": "resources/read", "params": M{"uri": uri}})
	if err != nil {
		t.Fatal(err)
	}
	request(t, server, string(body), &read)

	if read.Error != nil {
		return "", read.Error
	}
	if len(read.Result.Contents) != 1 || read.Result.Contents[0].Uri != uri {
		t.Fatalf("expected the contents of %s, got %v", uri, read.Result.Contents)
	}
	return read.Result.Contents[0].Text, nil
}

func TestResourceTemplatesAreListedApart(t *testing.T) {
	server := templateServer(t)

	var list struct {
		Result struct {
			Resources []map[string]any `json:"resources"`
		} `json:"result"`
	}
	request(t, server, `{"jsonrpc":"2.0","id":1,"method":"resources/list"}`, &list)
	if len(list.Result.Resources) != 1 || list.Result.Resources[0]["uri"] != "docs://guide" {
		t.Fatalf("expected only the guide as resource, got %v", list.Result.Resources)
	}

	var templates struct {
		Result struct {
			ResourceTemplates []map[string]any `json:"resourceTemplates"`
		} `json:"result"`
	}
	request(t, server, `{"jsonrpc":"2.0","id":1,"method":"resources/templates/list"}`, &templates)
	if len(templates.Result.ResourceTemplates) != 2 {
		t.Fatalf("expected two templates, got %v", templates.Result.ResourceTemplates)
	}
	template := templates.Result.ResourceTemplates[0]
	if template["uriTemplate"] != "docs://profiles/{id}/rules/{field}" || template["uri"] != nil {
		t.Fatalf("expected the template under uriTemplate, got %v", template)
	}
}

func TestResourcesReadPassesTheParamsOfATemplate(t *testing.T) {
	text, err := readText(t, templateServer(t), "docs://profiles/12/rules/work%20experience")
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if text != "12 work experience" {
		t.Fatalf("expected the decoded params, got %q", text)
	}
}

func TestResourcesReadPrefersAFixedUriOverATemplate(t *testing.T) {
	server := templateServer(t)
	AddResourceToServer(server, Resource{
		Uri:  "docs://profiles/12/rules/work",
		Name: "fixed",
		Read: func(_ context.Context, _ map[string]string) (string, error) {
			return "fixed", nil
		},
	})

	text, err := readText(t, server, "docs://profiles/12/rules/work")
	if err != nil || text != "fixed" {
		t.Fatalf("expected the fixed resource, got %q, %v", text, err)
	}
}

func TestResourcesReadTakesTheFirstTemplateThatMatches(t *testing.T) {
	text, err := readText(t, templateServer(t), "docs://profiles/latest/rules/work")
	if err != nil || text != "latest work" {
		t.Fatalf("expected the first template that was added, got %q, %v", text, err)
	}
}

func TestResourceTemplateParamsMatchOneSegment(t *testing.T) {
	_, err := readText(t, templateServer(t), "docs://profiles/12/13/rules/work")
	if err == nil || err.Code != resourceNotFoundCode {
		t.Fatalf("expected a resource not found error, got %v", err)
	}
}
