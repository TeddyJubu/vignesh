package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestClearPauseIfExpiredRestoresNotified(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	phone := "8801000000001"
	if _, err := db.GetOrCreateContact(phone); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateContact(phone, "", "{}", "notified"); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	_, err = db.db.Exec(
		`UPDATE contacts SET status = 'paused', status_before_pause = 'notified', paused_until = ? WHERE phone = ?`,
		past.UTC().Format(time.RFC3339), phone,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ClearPauseIfExpired(phone, time.Now()); err != nil {
		t.Fatal(err)
	}
	c, err := db.GetContact(phone)
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != "notified" {
		t.Fatalf("status = %q, want notified", c.Status)
	}
}
