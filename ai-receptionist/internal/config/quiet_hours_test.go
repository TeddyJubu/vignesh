package config

import (
	"testing"
	"time"
)

func TestQuietHoursOvernight(t *testing.T) {
	q := QuietHours{Enabled: true, TZ: "UTC", Start: "22:00", End: "08:00"}
	loc := time.UTC
	at23 := time.Date(2026, 5, 26, 23, 0, 0, 0, loc)
	at10 := time.Date(2026, 5, 26, 10, 0, 0, 0, loc)
	if !q.InQuietHours(at23) {
		t.Fatal("23:00 should be quiet")
	}
	if q.InQuietHours(at10) {
		t.Fatal("10:00 should be active")
	}
}

func TestQuietHoursDisabled(t *testing.T) {
	q := QuietHours{Enabled: false, Start: "22:00", End: "08:00"}
	if q.InQuietHours(time.Now()) {
		t.Fatal("disabled quiet hours")
	}
}
