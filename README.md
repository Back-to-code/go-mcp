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
server := mcp.NewServer("Example server")

type HelloWorldRequest struct {
	// github.com/invopop/jsonschema is used to generate the schema
	Name string `json:"name" jsonschema:"required" description:"The name of the person to greet"`
}

mcp.AddToolToServer(s, Tool{
	Name:        "hello_world",
	Description: "Example tool for server",
	Handler:     func(args HelloWorldRequest) (any, error) {
		return "Hello " + args.Name
	},
})

// Inside a http request to the mcp route
response := server.Handle(method /* string */, body /* []byte */)
fmt.Printf("%+v\n", response)
```
