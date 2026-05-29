package store

import "testing"

func TestInsertLeadContacts(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	jobID := "job-1"
	if err := db.InsertLeadContacts(jobID, []LeadContact{
		{Name: "A", Company: "Co", Email: "a@co.com", FitScore: 8, PitchAngle: "angle"},
		{Name: "B", Company: "Co2", Email: "b@co2.com", FitScore: 7},
	}); err != nil {
		t.Fatal(err)
	}
	n, err := db.CountLeadContactsByJob(jobID)
	if err != nil || n != 2 {
		t.Fatalf("count=%d err=%v", n, err)
	}
}
