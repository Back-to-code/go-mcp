# Go MCP

A go library for easially adding an mcp server to an exsisting server.

This library currently only supports the [Streamable Http](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http) standard.

## Public?

DO NOT MAKE THIS REPO PRIVATE!

This library is public so it can be easially imported by other private projects.
Once this library is private it's a fuckload of work to import this package.

Not only do we have go set go variables locally when we want to setup a project that makes use of this library but also in the pipelines and inside the docker containers.

## Usage?

```go
// Setup server
mcpServer := mcp.NewServer("Example server")

type HelloWorldRequest struct {
	// github.com/invopop/jsonschema is used to generate the schema
	Name string `json:"name" jsonschema:"required" description:"The name of the person to greet"`
}

mcp.AddToolToServer(mcpServer, mcp.Tool[HelloWorldRequest]{
	Name:        "hello_world",
	Description: "Example tool for server",
	Handler:     func(args HelloWorldRequest) (any, error) {
		return "Hello " + args.Name, nil
	},
})

// Inside a http request to the mcp route
response := mcpServer.Handle(method /* string */, body /* []byte */)
fmt.Printf("%+v\n", response)
```

### http server handler example

```go
http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
	// Read the body
	defer r.Body.Close()
	requestBody, _ := io.ReadAll(r.Body)

	// Let the mcp server handle the request
	response := mcpServer.Handle(r.Method, requestBody)

	// Respond with the payload of the mcp server
	w.Header().Set("Content-Type", response.ContentType)
	w.WriteHeader(response.Status)
	w.Write(response.Payload)
})
```
