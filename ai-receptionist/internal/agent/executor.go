package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type ToolResult struct {
	TaskName string `json:"task_name"`
	Tool     string `json:"tool"`
	Input    string `json:"input"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
	LatencyMS int64 `json:"latency_ms"`
}

func RunToolsParallel(ctx context.Context, tasks []AgentTask) []ToolResult {
	if len(tasks) == 0 {
		return nil
	}
	out := make([]ToolResult, len(tasks))
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for i := range tasks {
		i := i
		go func() {
			defer wg.Done()
			start := time.Now()
			r := ToolResult{
				TaskName: strings.TrimSpace(tasks[i].Name),
				Tool:     strings.TrimSpace(tasks[i].Tool),
				Input:    tasks[i].Input,
			}
			o, err := runStubTool(ctx, tasks[i])
			r.LatencyMS = time.Since(start).Milliseconds()
			if err != nil {
				r.Error = err.Error()
			} else {
				r.Output = o
			}
			out[i] = r
		}()
	}
	wg.Wait()
	return out
}

func runStubTool(ctx context.Context, t AgentTask) (string, error) {
	tool := strings.ToLower(strings.TrimSpace(t.Tool))
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	switch tool {
	case "check_calendar_availability":
		return `{"available":true,"slots":["tomorrow 3pm","tomorrow 5pm","fri 11am"]}`, nil
	case "collect_email":
		return `{"email":"missing"}`, nil
	case "align_time":
		return `{"timezone":"Asia/Singapore","suggested_slots":["tomorrow 3pm","fri 11am"]}`, nil
	case "book_appointment":
		return `{"booked":false,"reason":"stubbed_tool_no_side_effects"}`, nil
	case "":
		return "", fmt.Errorf("missing tool")
	default:
		return "", fmt.Errorf("unknown tool %q", t.Tool)
	}
}

