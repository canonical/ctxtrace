// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ctxtrace_test

import (
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
	w.WriteHeader(http.StatusOK)
}

func TestNewRoundTripper(t *testing.T) {
	c := qt.New(t)
	srv := httptest.NewServer(ctxtrace.Handler(http.HandlerFunc(dummyHandler)))
	defer srv.Close()

	client := NewTestClient(TestRoundTripper{r: ctxtrace.Transport{}, c: c})
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
	r.c.Assert(req.Header.Get(ctxtrace.TraceIDHeader), qt.Not(qt.IsNil))
	return r.r.RoundTrip(req)
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(rt http.RoundTripper) *http.Client {
	return &http.Client{
		Transport: rt,
	}
}
