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

// WithTraceField adds a trace-id zapctx field with a generated trace id. It
// returns the new context with the embedded trace id. This method is ideal
// to add the trace_id field when creating a new trace source, e.g., a cli
// command:
//  ctx := ctxtrace.WithTraceField(cmd.Context())
// This method sets the given context with a zap entry such as:
//  {"trace_id":"0ec8fad4-c77e-4631-9de7-6788e1b06770"}
// so it should not be used in a context that already has trace id. To set
// a specific trace id to the context use ContextWithTraceID.
func WithTraceField(ctx context.Context) context.Context {
	id := NewTraceID()
	ctx = ContextWithTraceID(ctx, id)
	return ctx
}

// ContextWithTraceID returns a context.Context with the given trace ID set.
// This method sets the given id into the trace_id context zap entry, so it
// is ideal when needs to transition from a non-traced context to a traced one.
// For example, when setting up HTTP handlers one usually may use something like
//  r.HandlerFunc(method, path, func(w http.ResponseWriter, r *http.Request) {
//		traceID := ctxtrace.TraceIDFromRequest(r)
//		ctx := ctxtrace.ContextWithTraceID(r.Context(), traceID)
//      ...
//  }
// to transition between the trace header from the incoming request to the
// internal context enriched with a trace_id entry.
func ContextWithTraceID(ctx context.Context, id string) context.Context {
	ctx = zapctx.WithFields(ctx, zap.String(traceIDCtx, id))
	return context.WithValue(ctx, traceIDContextKey{}, id)
}

// TraceIDFromContext returns the trace ID for a given context, or an empty string if not set.
func TraceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(traceIDContextKey{}).(string)
	return id
}

// TraceIDFromRequest returns the trace id from the request header or create a new one
// if it is empty. Similar to ContextWithTraceID this method fits well into a places
// where one needs to transition between an incoming request to some action based on
// the extracted (or new) trace id from the request header.
// For example, when setting up HTTP handlers one usually may use something like
//  r.HandlerFunc(method, path, func(w http.ResponseWriter, r *http.Request) {
//		traceID := ctxtrace.TraceIDFromRequest(r)
//		ctx := ctxtrace.ContextWithTraceID(r.Context(), traceID)
//      ...
//  }
// to extract the trace id from the request and inject it into a following context.
func TraceIDFromRequest(req *http.Request) string {
	existingTraceID := req.Header.Get(TraceIDHeader)
	if isValidTraceID(existingTraceID) {
		return existingTraceID
	}
	return NewTraceID()
}

// SetTraceHeader sets the given context trace id value into the given request
// header trace id. If the given context does not contain trace id value, this
// method will generate a new one then set it to the request header. This method
// is ideally placed into a http client that will transmit the internal context's
// trace id through the X-Trace-Id parameter to an external service. For instance
// it can be plugged into a httprequest.Doer as
//  type tracedDoer struct {
//		Doer httprequest.Doer
//  }
//
//  func (t tracedDoer) Do(req *http.Request) (*http.Response, error) {
//		ctx := req.Context()
//		ctxtrace.SetTraceHeader(ctx, req)
//		return t.Doer.Do(req)
//  }
// so it gets the request context from the caller client transforms it into a
// trace id header.
func SetTraceHeader(ctx context.Context, req *http.Request) {
	id := TraceIDFromContext(ctx)
	if !isValidTraceID(id) {
		id = NewTraceID()
	}
	req.Header.Set(TraceIDHeader, id)
}

// isValidTraceID returns true if the provided id is in the expected trace id format.
func isValidTraceID(id string) bool {
	if id == "" {
		return false
	}
	if IsTestingTraceID(id) {
		return true
	}
	_, err := uuid.Parse(id)
	return err == nil
}

// ContextWithTestingTraceID returns a context with the given trace ID set with a
// fixed testing prefix value indicating that this request should be considered for
// testing purposes only. This method could be used to discard testing requests
// from the auditing process.
func ContextWithTestingTraceID(ctx context.Context, id string) context.Context {
	return ContextWithTraceID(ctx, testingTraceIDPrefix+id)
}

// IsTestingTraceID returns whether a trace id is specially crafted for testing
// purposes only.
func IsTestingTraceID(id string) bool {
	return strings.HasPrefix(id, testingTraceIDPrefix)
}
