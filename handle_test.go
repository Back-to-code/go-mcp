package mcp

import (
	"context"
	"encoding/json"
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
