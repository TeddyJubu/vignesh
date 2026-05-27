package store

import "testing"

func TestAppSettings_UpsertAndGet(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.UpsertAppSetting("ai.provider", "openai"); err != nil {
		t.Fatal(err)
	}
	v, err := db.GetAppSetting("ai.provider")
	if err != nil {
		t.Fatal(err)
	}
	if v != "openai" {
		t.Fatalf("value=%q", v)
	}
}
