// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// Package ctxtrace provides tracing methods that simplify the task of
// keeping a trace id between HTTP clients and services by handling
// the conversion between HTTP header and context.Context.
package ctxtrace

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	TraceIDHeader        = "X-Trace-Id"
	testingTraceIDPrefix = "testing-"
)

type traceIDContextKey struct{}

// NewTraceID generates a new uuid v4 trace ID string.
func NewTraceID() string {
	return uuid.New().String()
}

// WithTraceID returns a context.Context with the given trace ID set.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDContextKey{}, id)
}

// TraceIDFromContext returns the trace ID for a given context, or an empty string if not set.
func TraceIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(traceIDContextKey{}).(string)
	return id
}

// TraceIDFromRequest returns the trace id from the request header or creates
// a new one if it is empty. Similar to ContextWithTraceID this method fits
// well into places where one needs to transition from an incoming request
// to some action based on the extracted (or new) trace id from the request header.
// For example, when setting up HTTP handlers one usually may use something like
//  r.HandlerFunc(method, path, func(w http.ResponseWriter, r *http.Request) {
//		traceID := ctxtrace.TraceIDFromRequest(r)
//		ctx := ctxtrace.ContextWithTraceID(r.Context(), traceID)
//      ...
//  }
// to extract the trace id from the request and inject it into a following context.
func TraceIDFromRequest(req *http.Request) string {
	existingTraceID := req.Header.Get(TraceIDHeader)
	if existingTraceID == "" {
		return NewTraceID()
	}
	return existingTraceID
}

// SetTraceHeader sets the given context trace id value into the given request
// header trace id. If the given context does not contain a trace id value, this
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
// so it gets the request context from the caller client and then transforms
// it into a trace id header.
func SetTraceHeader(ctx context.Context, req *http.Request) {
	id := TraceIDFromContext(ctx)
	if id == "" {
		id = NewTraceID()
	}
	req.Header.Set(TraceIDHeader, id)
}

// WithTestingTraceID returns a context with the given trace ID set with a fixed
// testing prefix value indicating that this request should be considered for
// testing purposes only. This method could be used to discard testing requests
// from the auditing process.
func WithTestingTraceID(ctx context.Context, id string) context.Context {
	return WithTraceID(ctx, testingTraceIDPrefix+id)
}

// IsTestingTraceID returns whether a trace id is specially crafted for testing
// purposes only.
func IsTestingTraceID(id string) bool {
	return strings.HasPrefix(id, testingTraceIDPrefix)
}

// Handler is a handler that get the trace id from the request, if empty generate one, put it in the context
// and set it on the response.
func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		traceID := TraceIDFromRequest(r)
		if traceID == "" {
			traceID = NewTraceID()
		}
		w.Header().Set(TraceIDHeader, traceID)
		h.ServeHTTP(w, r.WithContext(WithTraceID(r.Context(), traceID)))
	})
}
