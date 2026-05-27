package agent

import (
	"context"

	"ai-receptionist/internal/agent/tools"
)

type ToolResult struct {
	TaskName  string `json:"task_name"`
	Tool      string `json:"tool"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	LatencyMS int64  `json:"latency_ms"`
}

var defaultRunner = tools.NewRunner(tools.DefaultRegistry())

// SetDefaultRegistry replaces the global tool runner registry (tests).
func SetDefaultRegistry(reg *tools.Registry) {
	defaultRunner = tools.NewRunner(reg)
}

// RunToolsParallel executes planner agents via the tool registry.
func RunToolsParallel(ctx context.Context, rc tools.RunContext, tasks []AgentTask) []ToolResult {
	ctx = tools.ContextWithRun(ctx, rc)
	if rc.Deps.Calendar != nil {
		ctx = tools.ContextWithCalendar(ctx, rc.Deps.Calendar)
	}
	in := make([]tools.Task, len(tasks))
	for i, t := range tasks {
		in[i] = tools.Task{Name: t.Name, Tool: t.Tool, Input: t.Input}
	}
	raw := defaultRunner.RunParallel(ctx, rc, in)
	out := make([]ToolResult, len(raw))
	for i, r := range raw {
		out[i] = ToolResult{
			TaskName:  r.TaskName,
			Tool:      r.Tool,
			Input:     r.Input,
			Output:    r.Output,
			Error:     r.Error,
			LatencyMS: r.LatencyMS,
		}
	}
	return out
}
