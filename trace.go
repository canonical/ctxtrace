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
	// TraceIDHeader holds the header key that should be used when setting a
	// http.Request or http.ResponseWriter. It is used internally so when
	// using working with trace headers outside the library aiming to integrate
	// with it, use this constant.
	TraceIDHeader        = "X-Trace-Id"
	// Similar to TraceIDHeader, it holds the context key for logging as a reference
	// so that one does not need to replicate it on the different services using
	// trace in their logging context.
	TraceIDCtx           = "trace_id"
	testingTraceIDPrefix = "testing-"
)

type traceIDContextKey struct{}

// NewTraceID generates a new uuid v4 trace ID string.
func NewTraceID() string {
	return uuid.New().String()
}

// WithTraceID attaches the given ID to the given context. This ID will be
// used by the Transport to provide the X-Trace-Id header for requests
// containing a context with this value. If the given ID is "" then an new
// random ID will be created.
func WithTraceID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = NewTraceID()
	}
	return context.WithValue(ctx, traceIDContextKey{}, id)
}

// WithTestingTraceID attaches the given ID to the given context set with a fixed
// testing prefix value indicating that this request should be considered for
// testing purposes only. If the given ID is "" then an new random ID will be
// created. This ID will be used by the Transport to provide the X-Trace-Id
// header for requests containing a context with this value. This method could
// be used to discard testing requests from the auditing process.
func WithTestingTraceID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = NewTraceID()
	}
	// Avoid prepending it again if it is already there.
	if !strings.HasPrefix(id, testingTraceIDPrefix) {
		id = testingTraceIDPrefix + id
	}
	return WithTraceID(ctx, id)
}

// IsTestingTraceID returns whether a trace id is specially crafted for testing
// purposes only.
func IsTestingTraceID(id string) bool {
	return strings.HasPrefix(id, testingTraceIDPrefix)
}


// TraceIDFromContext returns the trace ID for a given context, or an empty
// string if not set.
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

// Transport implements http.RoundTripper. It transmits the trace id from the
// incoming request to the next one. If the incoming request header does not
// contain a trace id value, it will generate a new one then set it to the following
// request header.
// This RoundTipper is ideally placed into a http client that will transmit the
// internal context's trace id through the X-Trace-Id parameter to an external service:
//  client := &Client{
//		Params: p,
//		client: &http.Client{Transport: ctxtrace.Transport{}},
// 	}
// By default, Transport RoundTripper will be following by the http.DefaultTransport.
// An optional next http.RoundTripper can be given as Transport attribute so that it
// can be composed with other http.RoundTripper.
type Transport struct {
	// RoundTripper is an optional http.RoundTripper that will be called after
	// Transport.RoundTrip has enriched the incoming request with the trace id
	// header.
	RoundTripper http.RoundTripper
}

// RoundTrip implements http.RoundTripper interface to transfer the trace id, or
// creating a new one if it is empty, from the incoming request to the following
// http.RoundTripper. It is followed by http.DefaultTransport or a given RoundTripper
// when declaring Transport.
func (t Transport) RoundTrip(req *http.Request) (*http.Response, error) {
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
