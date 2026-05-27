package store

import (
	"testing"
)

func TestOpenCreatesAgentStates(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	err = db.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='agent_states'`).Scan(&name)
	if err != nil {
		t.Fatal(err)
	}
	if name != "agent_states" {
		t.Fatalf("table=%q", name)
	}
}
