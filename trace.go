// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

// Package ctxtrace provides tracing methods that simplify the task of
// keeping a trace id between HTTP clients and services by handling
// the conversion between HTTP header and context.Context.
package ctxtrace

import (
	"context"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

const (
	TraceIDHeader        = "X-Trace-Id"
	TraceIDCtx 			 = "trace_id"
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

// Handler is a handler that get the trace id from the request, if empty generate one,
// put it in the context and set it on the response.
func Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		traceID := traceIDFromRequest(r)
		w.Header().Set(TraceIDHeader, traceID)
		h.ServeHTTP(w, r.WithContext(WithTraceID(r.Context(), traceID)))
	})
}

// NewRoundTripper creates a http.RoundTripper that transmits the trace id
// from the incoming request to the next one. It is ideally placed
// when declaring a http.Client as
//  client := &Client{
//		Params: p,
//		client: &http.Client{Transport: ctxtrace.NewRoundTripper(nil)},
//	}
// An optional next http.RoundTripper can be given as parameter so that it
// can be composed with other http.RoundTripper.
func NewRoundTripper(tripper http.RoundTripper) http.RoundTripper {
	if tripper == nil {
		return &RoundTripper{r: http.DefaultTransport}
	}
	return &RoundTripper{r: tripper}
}

// RoundTripper implements http.RoundTripper.
type RoundTripper struct {
	r           http.RoundTripper
}

// RoundTrip implements http.Router interface to. This RoundTipper is ideally
// placed into a http client that will transmit the internal context's
// trace id through the X-Trace-Id parameter to an external service.
func (rt RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	setTraceHeader(req.Context(), req)
	return rt.r.RoundTrip(req)
}

// setTraceHeader sets the given context trace id value into the given request
// header trace id. If the given context does not contain a trace id value, this
// method will generate a new one then set it to the request header.
func setTraceHeader(ctx context.Context, req *http.Request) {
	id := TraceIDFromContext(ctx)
	if id == "" {
		id = NewTraceID()
	}
	req.Header.Set(TraceIDHeader, id)
}

// traceIDFromRequest returns the trace id from the request header or creates
// a new one if it is empty.
func traceIDFromRequest(req *http.Request) string {
	existingTraceID := req.Header.Get(TraceIDHeader)
	if existingTraceID == "" {
		return NewTraceID()
	}
	return existingTraceID
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
