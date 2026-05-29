package store

import (
	"testing"
	"time"
)

func TestAsyncJobLifecycle(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id, err := db.InsertAsyncJob(AsyncJob{
		ConvID:      "6590000001",
		JobType:     "scrape_leads",
		Payload:     `{"query":"dental clinics","count":5}`,
		NotifyOwner: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	j, err := db.GetAsyncJob(id)
	if err != nil || j == nil {
		t.Fatalf("get job: %v", err)
	}
	if j.Status != "pending" {
		t.Fatalf("expected pending, got %q", j.Status)
	}

	claimed, err := db.ListPendingAsyncJobs(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].ID != id {
		t.Fatalf("claim: %+v", claimed)
	}
	if claimed[0].Status != "running" {
		t.Fatalf("expected running after claim, got %q", claimed[0].Status)
	}

	if err := db.UpdateAsyncJobStatus(id, "completed", "ok", ""); err != nil {
		t.Fatal(err)
	}
	done, err := db.GetAsyncJob(id)
	if err != nil || done == nil {
		t.Fatal(err)
	}
	if done.Status != "completed" || done.Result != "ok" {
		t.Fatalf("unexpected done state: %+v", done)
	}
}

func TestResetStaleRunningJobs(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id, err := db.InsertAsyncJob(AsyncJob{JobType: "research_marketing", Payload: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.db.Exec(`UPDATE async_jobs SET status = 'running', updated_at = datetime('now', '-1 hour') WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}

	n, err := db.ResetStaleRunningJobs(30 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 reset, got %d", n)
	}
	j, _ := db.GetAsyncJob(id)
	if j.Status != "pending" {
		t.Fatalf("expected pending after reset, got %q", j.Status)
	}
}
