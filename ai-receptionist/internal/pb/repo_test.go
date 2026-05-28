package pb

import (
	"context"
	"testing"
)

func TestRepo_disabledNoOp(t *testing.T) {
	r := NewRepo(NewClient("", ""))
	ctx := context.Background()
	if err := r.UpsertSession(ctx, "1555", "general", "hi"); err != nil {
		t.Fatal(err)
	}
	id, err := r.InsertJob(ctx, "1555", "intent_classify", map[string]any{"intent": "general"})
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		t.Fatalf("id=%q want empty", id)
	}
	if err := r.UpdateJobStatus(ctx, "x", "done", nil, ""); err != nil {
		t.Fatal(err)
	}
}

func TestClient_Enabled(t *testing.T) {
	if NewClient("", "").Enabled() {
		t.Fatal("empty url should be disabled")
	}
	if !NewClient("http://127.0.0.1:8090", "tok").Enabled() {
		t.Fatal("expected enabled")
	}
}
