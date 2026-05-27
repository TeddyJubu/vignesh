package receptionist

import (
	"strings"
	"testing"

	"ai-receptionist/internal/config"
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

	cfg := &config.Config{BusinessName: "Acme Co", OwnerName: "Vignesh"}
	pb := NewPromptBuilder(cfg, db, "")
	out, err := pb.Build("6591234567", modeSales)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Julia") || !strings.Contains(out, "Acme Co") {
		t.Fatalf("missing baseline: %q", out)
	}
	if !strings.Contains(out, "## Soul") || !strings.Contains(out, "Epicware") {
		t.Fatalf("missing soul: %q", out)
	}
	if !strings.Contains(out, "## Client instructions") || !strings.Contains(out, "Universal rules") {
		t.Fatalf("missing client instructions: %q", out)
	}
	if !strings.Contains(out, "julia-sales") && !strings.Contains(out, "Mode runbook") {
		t.Fatalf("missing sales runbook: %q", out)
	}
}
