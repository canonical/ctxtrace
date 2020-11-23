package ctxtrace

import (
	"context"
	"net/http"
)

func TraceIDFromRequest(req *http.Request) string {
	return traceIDFromRequest(req)
}

func SetTraceHeader(ctx context.Context, req *http.Request) {
	setTraceHeader(ctx, req)
}