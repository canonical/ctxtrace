// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ctxtrace_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/canonical/ctxtrace"
)

func TestNewTraceID(t *testing.T) {
	c := qt.New(t)
	traceID := ctxtrace.NewTraceID()

	c.Assert(traceID, qt.Not(qt.IsNil))
}

func TestWithTraceIDEmpty(t *testing.T) {
	c := qt.New(t)
	ctx := ctxtrace.WithTraceID(context.Background(), "")
	traceID := ctxtrace.TraceIDFromContext(ctx)
	c.Assert(traceID, qt.Not(qt.Equals), "")
}

func TestWithTraceID(t *testing.T) {
	c := qt.New(t)
	traceID := "id"
	ctx := ctxtrace.WithTraceID(context.Background(), traceID)
	traceIDFromContext := ctxtrace.TraceIDFromContext(ctx)
	c.Assert(traceID, qt.Equals, traceIDFromContext)
}

func TestWithTestingTraceIDDoesNotPrependMultipleTimes(t *testing.T) {
	c := qt.New(t)
	traceID := "id"
	ctx := ctxtrace.WithTestingTraceID(context.Background(), traceID)
	ctx = ctxtrace.WithTestingTraceID(ctx, traceID)
	c.Assert(ctxtrace.TraceIDFromContext(ctx), qt.Satisfies, ctxtrace.IsTestingTraceID)
}

func TestWithTestingTraceIDEmpty(t *testing.T) {
	c := qt.New(t)
	ctx := ctxtrace.WithTestingTraceID(context.Background(), "")
	traceID := ctxtrace.TraceIDFromContext(ctx)
	c.Assert(traceID, qt.Satisfies, ctxtrace.IsTestingTraceID)
}

func TestHandlerWhenNoHeaderIsGiven(t *testing.T) {
	c := qt.New(t)
	srv := httptest.NewServer(ctxtrace.Handler(http.HandlerFunc(dummyHandler)))
	defer srv.Close()

	response, err := http.DefaultClient.Get(srv.URL)
	c.Assert(err, qt.IsNil)
	c.Assert(response.Header.Get(ctxtrace.TraceIDHeader), qt.Not(qt.IsNil))
}

func TestHandlerWhenHeaderIsGiven(t *testing.T) {
	c := qt.New(t)
	srv := httptest.NewServer(ctxtrace.Handler(http.HandlerFunc(dummyHandler)))
	defer srv.Close()

	request, err := http.NewRequest("GET", srv.URL, nil)
	c.Assert(err, qt.IsNil)
	traceID := ctxtrace.NewTraceID()
	request.Header.Set(ctxtrace.TraceIDHeader, traceID)

	response, err := http.DefaultClient.Do(request)
	c.Assert(err, qt.IsNil)
	c.Assert(response.Header.Get(ctxtrace.TraceIDHeader), qt.Equals, traceID)
}

func dummyHandler(w http.ResponseWriter, r *http.Request) {
	// The default behavior is w.WriteHeader(http.StatusOK)
}

func TestNewRoundTripper(t *testing.T) {
	c := qt.New(t)
	srv := httptest.NewServer(ctxtrace.Handler(http.HandlerFunc(dummyHandler)))
	defer srv.Close()

	client := http.Client{Transport: TestRoundTripper{r: ctxtrace.Transport{}, c: c}}
	_, err := client.Get(srv.URL)
	c.Assert(err, qt.IsNil)
}

// TestRoundTripper wraps around the Transport we are testing and asserts
// for the presence of a trace id in the request.
type TestRoundTripper struct {
	r http.RoundTripper
	c *qt.C
}

func (r TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.c.Check(req.Header.Get(ctxtrace.TraceIDHeader), qt.Not(qt.IsNil))
	return r.r.RoundTrip(req)
}
