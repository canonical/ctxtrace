// Copyright 2020 Canonical Ltd.
// Licensed under the LGPLv3, see LICENCE file for details.

package ctxtrace

var TraceIdKey = traceIDContextKey{}

func IsValidTraceID(id string) bool {
	return isValidTraceID(id)
}
