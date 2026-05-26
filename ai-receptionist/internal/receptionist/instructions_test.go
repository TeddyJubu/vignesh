package receptionist

import (
	"strings"
	"testing"

	"ai-receptionist/internal/store"
)

func TestBuildAgentInstructions_IncludesSoulAndClient(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	soul, err := db.GetAgentNote("identity_soul")
	if err != nil {
		t.Fatal(err)
	}
	if soul == "" {
		t.Fatal("expected seeded identity_soul")
	}

	out, err := buildAgentInstructions(db, "6591234567", "## Client\nTest rule\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Julia") {
		t.Fatalf("missing baseline: %q", out)
	}
	if !strings.Contains(out, "## Soul") || !strings.Contains(out, "Vignesh") {
		t.Fatalf("missing soul: %q", out)
	}
	if !strings.Contains(out, "## Client instructions") || !strings.Contains(out, "Test rule") {
		t.Fatalf("missing client instructions: %q", out)
	}
}
