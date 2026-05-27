package agent

import (
	"context"
	"testing"
	"time"

	"ai-receptionist/internal/agent/tools"
)

func TestRunToolsParallel_FanoutAndCollect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tasks := []AgentTask{
		{Name: "calendar", Tool: "check_calendar_availability", Input: "next week"},
		{Name: "tz", Tool: "align_time", Input: "SG"},
	}
	rc := tools.RunContext{ConvID: "6590000000"}
	out := RunToolsParallel(ctx, rc, tasks)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Tool != "check_calendar_availability" || out[0].Error != "" || out[0].Output == "" {
		t.Fatalf("out[0]=%+v", out[0])
	}
	if out[1].Tool != "align_time" || out[1].Error != "" || out[1].Output == "" {
		t.Fatalf("out[1]=%+v", out[1])
	}
}

func TestRunToolsParallel_UnknownTool(t *testing.T) {
	ctx := context.Background()
	out := RunToolsParallel(ctx, tools.RunContext{ConvID: "6590000000"}, []AgentTask{{Name: "x", Tool: "does_not_exist", Input: "a"}})
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Error == "" {
		t.Fatalf("expected error, got %+v", out[0])
	}
}
