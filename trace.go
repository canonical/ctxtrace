// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// Package tracectx provides tracing methods that easy the task of
// keeping a trace id between HTTP client and services by handling
// the conversion between HTTP header and zapctx tracing into context.Context.

package ctxtrace

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/juju/zaputil/zapctx"
	"go.uber.org/zap"
)

const (
	TraceIDHeader        = "X-Trace-Id"
	traceIDCtx           = "trace_id"
	testingTraceIDPrefix = "testing-"
)

type traceIDContextKey struct{}

// NewTraceID generates a new uuid v4 trace ID string.
func NewTraceID() string {
	return uuid.New().String()
}

// NewTracedContext adds a trace-id zapctx field with a generated trace id. It
// returns the new context with the embedded trace id.
func NewTracedContext(ctx context.Context) context.Context {
	id := NewTraceID()
	ctx = withZap(ctx, id)
	ctx = ContextWithTraceID(ctx, id)
	return ctx
}

// TracedContext adds a trace-id zapctx field with a given trace id or generate
// a new one. It returns the new context with the embedded trace id.
func TracedContext(ctx context.Context, id string) context.Context {
	if !IsValidTraceID(id) {
		return NewTracedContext(ctx)
	}
	ctx = withZap(ctx, id)
	ctx = ContextWithTraceID(ctx, id)
	return ctx
}

// withZap returns a context.Context with the given trace ID field set.
func withZap(ctx context.Context, id string) context.Context {
	return zapctx.WithFields(ctx, zap.String(traceIDCtx, id))
}

// ContextWithTraceID returns a context.Context with the given trace ID set.
func ContextWithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDContextKey{}, id)
}

// WithTestingTraceID returns a context with the given trace ID set with a
// fixed prefix value indicating that this request should not considered
// for auditing purposes.
func ContextWithTestingTraceID(ctx context.Context, id string) context.Context {
	return ContextWithTraceID(ctx, testingTraceIDPrefix+id)
}

// IsTestingTraceID
func IsTestingTraceID(id string) bool {
	return strings.HasPrefix(id, testingTraceIDPrefix)
}

// IsValidTraceID returns true if the provided id is in the expected trace id format.
func IsValidTraceID(id string) bool {
	if id == "" {
		return false
	}
	if IsTestingTraceID(id) {
		return true
	}
	_, err := uuid.Parse(id)
	return err == nil
}

// TraceIDFromContext returns the trace ID for a given context, or an empty string if not set.
func TraceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(traceIDContextKey{}).(string)
	return id
}

// HTTPTracedContext sets up the trace header from the given http.Request into the given
// response http.ResponseWriter. It returns a context properly setup with the zapctx entry
// containing the given request trace-id. This method is useful when defining handler as it
// will make the necessary transition between the trace header from the incoming request
// to the output context trace-id. For example it fits to a setup such as:
//  httprouter.Router.HandlerFunc(method, path, func(rw http.ResponseWriter, r *http.Request) {
//	 ctx := tracectx.HTTPTracedContext(rw, r)
//   ...
//  }
func HTTPTracedContext(rw http.ResponseWriter, req *http.Request) context.Context {
	id := ensureTraceHeader(req)
	rw.Header().Set(TraceIDHeader, id)
	return TracedContext(req.Context(), id)
}

func ensureTraceHeader(req *http.Request) string {
	existingTraceID := req.Header.Get(TraceIDHeader)
	if IsValidTraceID(existingTraceID) {
		return existingTraceID
	}
	return NewTraceID()
}

// SetTraceHeader sets the given context trace id value into the given request
// header trace id. If the given context does not trace id value, this method will generate
// a new one then set it to the request header.
func SetTraceHeader(ctx context.Context, req *http.Request) {
	id := TraceIDFromContext(ctx)
	if !IsValidTraceID(id) {
		id = NewTraceID()
	}
	req.Header.Set(TraceIDHeader, id)
}
