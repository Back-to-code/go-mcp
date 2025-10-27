package main

// Can be ran using:
// go run example/main.go

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"bitbucket.org/teamscript/go-mcp"
)

func main() {
	mcpServer := mcp.NewServer("Example server")

	type HelloWorldRequest struct {
		// github.com/invopop/jsonschema is used to generate the schema
		Name string `json:"name" jsonschema:"required" description:"The name of the person to greet"`
	}
	mcp.AddToolToServer(mcpServer, mcp.Tool[HelloWorldRequest]{
		Name:        "hello_world",
		Description: "Example tool for server",
		Handler: func(args HelloWorldRequest, _ context.Context) (any, error) {
			return "Hello " + args.Name, nil
		},
	})

	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Read the body
		defer r.Body.Close()
		requestBody, _ := io.ReadAll(r.Body)

		// Let the mcp server handle the request
		response := mcpServer.Handle(r.Method, requestBody, context.Background())

		// Respond with the payload of the mcp server
		w.Header().Set("Content-Type", response.ContentType)
		w.WriteHeader(response.Status)
		w.Write(response.Payload)
	})

	fmt.Println("serving on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
