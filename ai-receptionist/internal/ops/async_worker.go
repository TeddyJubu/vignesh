package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"
)

// JobHandler processes one async job type.
type JobHandler func(ctx context.Context, job store.AsyncJob) (result string, err error)

// AsyncWorker polls pending jobs and runs handlers.
type AsyncWorker struct {
	Store    *store.DB
	Cfg      *config.Config
	WA       *whatsapp.Client
	Handlers map[string]JobHandler
	Interval time.Duration
}

func (w *AsyncWorker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = 30 * time.Second
	}
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *AsyncWorker) tick(ctx context.Context) {
	jobs, err := w.Store.ListPendingAsyncJobs(5)
	if err != nil {
		log.Printf("async_worker: list: %v", err)
		return
	}
	for _, job := range jobs {
		w.processOne(ctx, job)
	}
}

func (w *AsyncWorker) processOne(ctx context.Context, job store.AsyncJob) {
	h := w.Handlers[strings.ToLower(job.JobType)]
	if h == nil {
		_ = w.Store.UpdateAsyncJobStatus(job.ID, "failed", "", "unknown job type")
		return
	}
	_ = w.Store.UpdateAsyncJobStatus(job.ID, "running", "", "")
	result, err := h(ctx, job)
	if err != nil {
		_ = w.Store.UpdateAsyncJobStatus(job.ID, "failed", result, err.Error())
		if job.NotifyOwner && w.WA != nil && w.Cfg != nil {
			msg := fmt.Sprintf("Job %s (%s) failed: %s", job.JobType, job.ID, err.Error())
			_ = whatsapp.SendText(ctx, w.WA, whatsapp.PhoneToJID(w.Cfg.OwnerNumber), msg)
		}
		return
	}
	_ = w.Store.UpdateAsyncJobStatus(job.ID, "completed", result, "")
	if job.NotifyOwner && w.WA != nil && w.Cfg != nil && strings.TrimSpace(result) != "" {
		summary := result
		if len(summary) > 500 {
			summary = summary[:500] + "…"
		}
		msg := fmt.Sprintf("Job %s done:\n%s", job.JobType, summary)
		_ = whatsapp.SendText(ctx, w.WA, whatsapp.PhoneToJID(w.Cfg.OwnerNumber), msg)
	}
}

// ParseJobPayload unmarshals job payload JSON.
func ParseJobPayload(job store.AsyncJob, dst any) error {
	if strings.TrimSpace(job.Payload) == "" || job.Payload == "{}" {
		return nil
	}
	return json.Unmarshal([]byte(job.Payload), dst)
}
