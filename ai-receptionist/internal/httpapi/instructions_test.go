package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
)

func TestHandleInstructions_GET_shape(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	cfg := &config.Config{BusinessName: "Epicware", OwnerName: "Vignesh"}
	srv := New(cfg, db, "", "")
	srv.SetPromptMaterials("prompt {{business_name}}", "", "")

	req := httptest.NewRequest(http.MethodGet, "/api/instructions", nil)
	rec := httptest.NewRecorder()
	srv.handleInstructions(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var out instructionsPayload
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.IdentitySoul, "Julia") {
		t.Fatalf("missing soul: %q", out.IdentitySoul[:min(80, len(out.IdentitySoul))])
	}
	if out.ClientInstructions.Content == "" {
		t.Fatal("expected knowledge base content")
	}
	if !strings.Contains(out.Preview, "SYSTEM") || !strings.Contains(out.Preview, "USER TURN") {
		t.Fatalf("preview missing bundled sections: %q", out.Preview[:min(200, len(out.Preview))])
	}
	if out.PromptLayout != "bundled" {
		t.Fatalf("layout %q", out.PromptLayout)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
