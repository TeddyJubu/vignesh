package tools

import (
	"context"
	"testing"
)

func TestRunner_runOne_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := NewRunner(DefaultRegistry())
	res := r.runOne(ctx, RunContext{ConvID: "6590000000"}, Task{Name: "x", Tool: "align_time", Input: "a"})
	if res.Error == "" {
		t.Fatalf("expected error, got %+v", res)
	}
}
