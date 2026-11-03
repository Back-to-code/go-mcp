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
		Read: func(_ context.Context) (string, error) {
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
		Read: func(_ context.Context) (string, error) {
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
	read := func(_ context.Context) (string, error) { return "", nil }

	cases := map[string]Resource{
		"no uri":       {Name: "guide", Read: read},
		"no name":      {Uri: "docs://other", Read: read},
		"no read":      {Uri: "docs://other", Name: "other"},
		"uri is taken": {Uri: "docs://guide", Name: "guide", Read: read},
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
