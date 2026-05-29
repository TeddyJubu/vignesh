package ops

import (
	"context"
	"testing"
	"time"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
)

func TestAsyncWorkerProcessesJob(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id, err := db.InsertAsyncJob(store.AsyncJob{
		JobType: "dispatch_webhook",
		Payload: `{"url":"http://127.0.0.1:1","body":"{}"}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	w := &AsyncWorker{
		Store:    db,
		Cfg:      &config.Config{OwnerNumber: "6590000001"},
		Handlers: DefaultJobHandlers(WorkerEnv{}),
		Interval: time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	w.tick(ctx)

	j, err := db.GetAsyncJob(id)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	if j.Status != "completed" && j.Status != "failed" {
		t.Fatalf("expected terminal status, got %q", j.Status)
	}
}
