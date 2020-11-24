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
	TraceIDCtx           = "trace_id"
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

// Handler is a handler that get the trace id from the request, if empty generate
// a new one, put it in the context and set it on the response.
func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get(TraceIDHeader)
		if traceID == "" {
			traceID = NewTraceID()
		}
		w.Header().Set(TraceIDHeader, traceID)
		h.ServeHTTP(w, r.WithContext(WithTraceID(r.Context(), traceID)))
	})
}

// Transport implements http.RoundTripper.
type Transport struct {
	RoundTripper http.RoundTripper
}

// RoundTrip implements http.Router interface to. It that transmits the trace id
// from the incoming request to the next one. If the incoming request header does
// not contain a trace id value, this method will generate a new one then set it
// to the following request header. This RoundTipper is ideally placed
// into a http client that will transmit the internal context's trace id through
// the X-Trace-Id parameter to an external service.
//  client := &Client{
//		Params: p,
//		client: &http.Client{Transport: ctxtrace.Transport{}},
// 	}
// An optional next http.RoundTripper can be given as Transport attribute
// so that it can be composed with other http.RoundTripper.
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Make the zero value useful.
	rt := t.RoundTripper
	if rt == nil {
		rt = http.DefaultTransport
	}

	if req.Header.Get(TraceIDHeader) != "" {
		// If the request already has a trace header don't overwrite it.
		return rt.RoundTrip(req)
	}

	newReq := *req
	newReq.Header = make(http.Header, len(req.Header) + 1)
	// Copy headers from old to the new request.
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	traceID := TraceIDFromContext(newReq.Context())
	if traceID == "" {
		traceID = NewTraceID()
	}
	newReq.Header.Set(TraceIDHeader, traceID)
	return rt.RoundTrip(&newReq)
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
