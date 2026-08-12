package mcp

import "context"

type contextKey int

const toolNameContextKey contextKey = iota

// contextWithToolName returns a copy of ctx that carries the name of the tool
// that is being called.
func contextWithToolName(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, toolNameContextKey, name)
}

// ToolNameFromContext returns the name of the tool the handler was called for.
// The second return value is false when ctx does not originate from a tool
// handler.
func ToolNameFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(toolNameContextKey).(string)
	return name, ok
}
