package session

import (
	"testing"
	"time"

	"ai-receptionist/internal/store"
)

func TestFormatLastTurnsForPrompt_orderAndLimit(t *testing.T) {
	msgs := []store.Message{
		{Role: "user", Message: "first"},
		{Role: "assistant", Message: "reply one"},
		{Role: "system", Message: "ignored"},
		{Role: "user", Message: "second"},
		{Role: "assistant", Message: "reply two"},
		{Role: "user", Message: "third"},
	}
	got := FormatLastTurnsForPrompt(msgs, 3)
	want := "user: second\nassistant: reply two\nuser: third"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFormatLastTurnsForPrompt_empty(t *testing.T) {
	if got := FormatLastTurnsForPrompt(nil, 5); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestGetLastTurns_delegatesToStore(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	now := time.Now()
	_ = now
	if err := db.InsertMessage("15551234567", "user", "hello"); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertMessage("15551234567", "assistant", "hi there"); err != nil {
		t.Fatal(err)
	}

	msgs, err := GetLastTurns(t.Context(), db, "15551234567", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || msgs[0].Message != "hello" || msgs[1].Message != "hi there" {
		t.Fatalf("msgs=%+v", msgs)
	}
}
