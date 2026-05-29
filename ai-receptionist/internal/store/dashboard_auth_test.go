package store

import (
	"testing"
	"time"
)

func TestDashboardOTPAndSession(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	phone := "900"
	if err := db.UpsertAccessRole(phone, "manager", map[string]bool{"settings": true}); err != nil {
		t.Fatalf("upsert role: %v", err)
	}

	code, _, err := db.CreateDashboardOTP(phone, 2*time.Minute)
	if err != nil {
		t.Fatalf("create otp: %v", err)
	}

	ok, err := db.VerifyDashboardOTP(phone, "000000")
	if err != nil {
		t.Fatalf("verify otp wrong: %v", err)
	}
	if ok {
		t.Fatalf("expected wrong code to fail")
	}

	ok, err = db.VerifyDashboardOTP(phone, code)
	if err != nil {
		t.Fatalf("verify otp: %v", err)
	}
	if !ok {
		t.Fatalf("expected code to verify")
	}

	token, sess, err := db.CreateDashboardSession(phone, 24*time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if token == "" || sess == nil || sess.Role != "manager" {
		t.Fatalf("unexpected session: token=%q sess=%+v", token, sess)
	}

	got, err := db.GetDashboardSessionByToken(token)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got == nil || got.Phone != phone || got.Role != "manager" || !got.Permissions["settings"] {
		t.Fatalf("unexpected got: %+v", got)
	}

	if err := db.RevokeDashboardSession(token); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	got, err = db.GetDashboardSessionByToken(token)
	if err != nil {
		t.Fatalf("get after revoke: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil after revoke, got: %+v", got)
	}
}

