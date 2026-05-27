package store

import "testing"

func TestInsertDreamProposal(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id := "dream-test-1"
	if err := db.InsertDreamProposal(DreamProposal{
		ID:        id,
		Status:    "proposed",
		Title:     "Test",
		Patch:     `{"target_key":"identity_soul","new_content":"hello"}`,
		Rationale: "unit test",
	}); err != nil {
		t.Fatal(err)
	}
	p, err := db.GetDreamProposal(id)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || p.Title != "Test" || p.Status != "proposed" {
		t.Fatalf("got %+v", p)
	}
}
