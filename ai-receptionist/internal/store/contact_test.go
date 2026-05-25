package store

import (
	"testing"
	"time"
)

func TestContactIsPaused(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	c := &Contact{Status: "paused", PausedUntil: &future}
	if !c.IsPaused(time.Now()) {
		t.Fatal("expected paused")
	}

	c.PausedUntil = &past
	if c.IsPaused(time.Now()) {
		t.Fatal("expected expired pause")
	}

	c.PausedUntil = nil
	if !c.IsPaused(time.Now()) {
		t.Fatal("paused without until should stay paused")
	}
}
