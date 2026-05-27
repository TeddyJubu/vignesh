package tools

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	defaultToolTimeout = 8 * time.Second
	maxPlannerAgents   = 4
	maxConcurrency     = 4
)

// Task is a single planner agent invocation.
type Task struct {
	Name  string
	Tool  string
	Input string
}

// Result is the outcome of a tool run.
type Result struct {
	TaskName  string
	Tool      string
	Input     string
	Output    string
	Error     string
	LatencyMS int64
}

// Runner executes registry tools with timeouts and audit logging.
type Runner struct {
	Registry *Registry
	Timeout  time.Duration
}

func NewRunner(reg *Registry) *Runner {
	return &Runner{Registry: reg, Timeout: defaultToolTimeout}
}

func (r *Runner) RunParallel(ctx context.Context, rc RunContext, tasks []Task) []Result {
	if len(tasks) == 0 {
		return nil
	}
	if len(tasks) > maxPlannerAgents {
		tasks = tasks[:maxPlannerAgents]
	}
	out := make([]Result, len(tasks))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for i := range tasks {
		i := i
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = r.runOne(ctx, rc, tasks[i])
		}()
	}
	wg.Wait()
	return out
}

func (r *Runner) runOne(ctx context.Context, rc RunContext, t Task) Result {
	start := time.Now()
	res := Result{
		TaskName: strings.TrimSpace(t.Name),
		Tool:     strings.TrimSpace(t.Tool),
		Input:    t.Input,
	}
	toolName := strings.ToLower(strings.TrimSpace(t.Tool))
	tool, ok := r.Registry.Get(toolName)
	if !ok {
		res.Error = "unknown tool"
		res.LatencyMS = time.Since(start).Milliseconds()
		r.audit(rc, res)
		return res
	}
	toolCtx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	output, err := tool.Run(toolCtx, t.Input)
	res.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = err.Error()
	} else {
		res.Output = output
	}
	r.audit(rc, res)
	return res
}

func (r *Runner) audit(rc RunContext, res Result) {
	if rc.Deps.Store == nil {
		return
	}
	_ = rc.Deps.Store.InsertToolRun(rc.ConvID, res.Tool, res.Input, res.Output, res.Error, res.LatencyMS)
}
