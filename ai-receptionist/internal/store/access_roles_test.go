package store

import "testing"

func TestAccessRolesCRUD(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Default admin seed should exist.
	admins, err := db.ListAccessRoles("admin", 100)
	if err != nil {
		t.Fatalf("list admins: %v", err)
	}
	if len(admins) == 0 {
		t.Fatalf("expected seeded admin")
	}

	phone := "12345"
	if err := db.UpsertAccessRole(phone, "manager", map[string]bool{"settings": true}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := db.GetAccessRole(phone)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.Role != "manager" || !got.Permissions["settings"] {
		t.Fatalf("unexpected role: %+v", got)
	}
	if err := db.DeleteAccessRole(phone); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = db.GetAccessRole(phone)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil role after delete, got: %+v", got)
	}
}

func TestEnsureClientRoleDoesNotOverrideManager(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	phone := "777"
	if err := db.UpsertAccessRole(phone, "manager", map[string]bool{"instructions": true}); err != nil {
		t.Fatalf("upsert manager: %v", err)
	}
	if err := db.EnsureClientRole(phone); err != nil {
		t.Fatalf("ensure client: %v", err)
	}
	got, err := db.GetAccessRole(phone)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil || got.Role != "manager" {
		t.Fatalf("expected manager to remain, got: %+v", got)
	}
}

