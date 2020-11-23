// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ctxtrace_test

import (
	"context"
	"net/http"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/canonical/ctxtrace"
)

func TestNewTraceID(t *testing.T) {
	c := qt.New(t)
	traceID := ctxtrace.NewTraceID()

	c.Assert(traceID, qt.Not(qt.IsNil))
}

func TestSetTraceHeaderFromContext(t *testing.T) {
	c := qt.New(t)
	type args struct {
		ctx context.Context
		req *http.Request
	}
	dummyRequest, _ := http.NewRequest("GET", "https://example.com/path", nil)
	tests := []struct {
		name string
		args args
	}{{
		name: "from empty context",
		args: args{
			ctx: context.Background(),
			req: dummyRequest,
		}}, {
		name: "from existing context trace id",
		args: args{
			ctx: ctxtrace.WithTraceID(context.Background(), ctxtrace.NewTraceID()),
			req: dummyRequest,
		}},
	}
	for _, tt := range tests {
		ctxtrace.SetTraceHeader(tt.args.ctx, tt.args.req)
		c.Assert(tt.args.req.Header.Get(ctxtrace.TraceIDHeader), qt.Not(qt.IsNil))
	}
}

func TestTraceIDFromRequest(t *testing.T) {
	c := qt.New(t)

	dummyRequest, _ := http.NewRequest("GET", "https://example.com/path", nil)
	traceID := ctxtrace.NewTraceID()
	dummyRequest.Header.Set(ctxtrace.TraceIDHeader, traceID)

	requestTraceID := ctxtrace.TraceIDFromRequest(dummyRequest)
	c.Assert(requestTraceID, qt.Equals, traceID)
}

func TestTraceIDFromRequestWithEmptyID(t *testing.T) {
	c := qt.New(t)

	dummyRequest, _ := http.NewRequest("GET", "https://example.com/path", nil)

	requestTraceID := ctxtrace.TraceIDFromRequest(dummyRequest)
	c.Assert(requestTraceID, qt.Not(qt.IsNil))
}
