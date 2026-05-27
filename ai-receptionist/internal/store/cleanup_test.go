package store

import (
	"encoding/json"
	"testing"
	"time"

	"ai-receptionist/internal/agent"
)

func TestPurgeStaleAgentStates(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	old := agent.State{Plan: agent.Plan{Goal: "x"}, StartedAtUNIX: time.Now().Add(-2 * time.Hour).Unix()}
	if err := db.UpsertAgentState("old", old); err != nil {
		t.Fatalf("upsert old: %v", err)
	}
	if _, err := db.GetAgentState("old"); err != nil {
		t.Fatalf("get old after upsert: %v", err)
	}
	fresh := agent.State{Plan: agent.Plan{Goal: "y"}, StartedAtUNIX: time.Now().Unix()}
	if err := db.UpsertAgentState("fresh", fresh); err != nil {
		t.Fatal(err)
	}
	n, err := db.PurgeStaleAgentStates(time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("removed=%d", n)
	}
	if _, err := db.GetAgentState("old"); err == nil {
		t.Fatal("expected old gone")
	}
	if _, err := db.GetAgentState("fresh"); err != nil {
		t.Fatal("expected fresh kept")
	}
}

func TestCapMessagesPerContact(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	phone := "6591111111"
	for i := 0; i < 5; i++ {
		if err := db.InsertMessage(phone, "user", "m"); err != nil {
			t.Fatal(err)
		}
	}
	n, err := db.CapMessagesPerContact(3)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("trimmed=%d", n)
	}
	msgs, err := db.RecentMessages(phone, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("len=%d", len(msgs))
	}
}

func TestPurgeStaleAgentStates_JSON(t *testing.T) {
	var st agent.State
	b, _ := json.Marshal(st)
	if len(b) == 0 {
		t.Fatal("marshal")
	}
	_ = b
}
