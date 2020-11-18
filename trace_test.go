// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ctxtrace_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/juju/zaputil/zapctx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	ctxtrace "github.com/CanonicalLtd/tracectx"
)

func TestNewTraceID(t *testing.T) {
	c := qt.New(t)
	traceID := ctxtrace.NewTraceID()

	c.Assert(traceID, qt.Satisfies, ctxtrace.IsValidTraceID)
}

func TestWithTraceField(t *testing.T) {
	ctx := context.Background()

	var buffer bytes.Buffer
	logger := newLogger(&buffer)
	ctx = zapctx.WithLogger(ctx, logger)

	c := qt.New(t)
	tracedContext := ctxtrace.WithTraceField(ctx)

	traceID, ok := tracedContext.Value(ctxtrace.TraceIdKey).(string)
	c.Assert(ok, qt.IsTrue)
	c.Assert(traceID, qt.Satisfies, ctxtrace.IsValidTraceID)

	zapctx.Logger(tracedContext).Info("")
	c.Assert(buffer.String(), qt.Contains, traceID)
}

func TestIsValidTraceID(t *testing.T) {
	c := qt.New(t)
	tests := []struct {
		name  string
		id    string
		valid bool
	}{{
		name: "empty id",
		id:   "",
	}, {
		name: "invalid id",
		id:   "asdasdasdas",
	}, {
		name:  "valid id",
		id:    ctxtrace.NewTraceID(),
		valid: true,
	}}
	for _, test := range tests {
		c.Assert(ctxtrace.IsValidTraceID(test.id), qt.Equals, test.valid)
	}
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
			ctx: ctxtrace.ContextWithTraceID(context.Background(), ctxtrace.NewTraceID()),
			req: dummyRequest,
		}},
	}
	for _, tt := range tests {
		ctxtrace.SetTraceHeader(tt.args.ctx, tt.args.req)
		c.Assert(ctxtrace.IsValidTraceID(tt.args.req.Header.Get(ctxtrace.TraceIDHeader)), qt.IsTrue)
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

func newLogger(w io.Writer) *zap.Logger {
	config := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,
	}
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config),
		zapcore.AddSync(w),
		zapcore.InfoLevel,
	)
	return zap.New(core)
}

func TestTraceIDFromRequestWithEmptyID(t *testing.T) {
	c := qt.New(t)

	dummyRequest, _ := http.NewRequest("GET", "https://example.com/path", nil)

	requestTraceID := ctxtrace.TraceIDFromRequest(dummyRequest)
	c.Assert(requestTraceID, qt.Satisfies, ctxtrace.IsValidTraceID)
}
