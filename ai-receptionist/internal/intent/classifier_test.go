package intent

import (
	"encoding/json"
	"testing"

	"ai-receptionist/internal/ai"
)

func TestDecodeResult_valid(t *testing.T) {
	raw := `{"intent":"support","confidence":0.91,"summary":"pricing question"}`
	r, err := decodeResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := normalizeResult(r)
	if out.Intent != "support" || out.Confidence != 0.91 || out.Summary != "pricing question" {
		t.Fatalf("%+v", out)
	}
}

func TestDecodeResult_stripsFences(t *testing.T) {
	raw := "```json\n{\"intent\":\"sales_qualify\",\"confidence\":0.8,\"summary\":\"book a call\"}\n```"
	r, err := decodeResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Intent != "sales_qualify" {
		t.Fatalf("intent=%q", r.Intent)
	}
}

func TestNormalizeResult_unknownIntent(t *testing.T) {
	out := normalizeResult(Result{Intent: "unknown_thing", Confidence: 1.5, Summary: "x"})
	if out.Intent != "general" {
		t.Fatalf("intent=%q", out.Intent)
	}
	if out.Confidence != 1 {
		t.Fatalf("conf=%v", out.Confidence)
	}
}

func TestEchoLine(t *testing.T) {
	got := EchoLine(Result{Intent: "general", Confidence: 0.5, Summary: "hello"})
	want := "intent=general conf=0.50 summary=hello"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestDecodeResult_invalidJSON(t *testing.T) {
	_, err := decodeResult("{not json")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyIntentHints_pricing(t *testing.T) {
	got := applyIntentHints("What are your pricing plans?", Result{Intent: "sales_qualify", Confidence: 0.9, Summary: "pricing"})
	if got.Intent != "support" {
		t.Fatalf("intent=%q", got.Intent)
	}
}

func TestApplyIntentHints_bookMeeting(t *testing.T) {
	got := applyIntentHints("I want to book a meeting", Result{Intent: "outbound_book", Confidence: 0.9, Summary: "book"})
	if got.Intent != "sales_qualify" {
		t.Fatalf("intent=%q", got.Intent)
	}
}

func TestApplyIntentHints_tomorrow(t *testing.T) {
	got := applyIntentHints("What am I doing tomorrow?", Result{Intent: "general", Confidence: 0.5, Summary: "schedule"})
	if got.Intent != "calendar_check" {
		t.Fatalf("intent=%q", got.Intent)
	}
}

func TestRepairIntentPrompt_roundTrip(t *testing.T) {
	msgs := ai.RepairIntentPrompt(`{broken`)
	if len(msgs) != 2 {
		t.Fatalf("len=%d", len(msgs))
	}
	var probe map[string]any
	_ = json.Unmarshal([]byte(`{"intent":"general","confidence":0,"summary":"ok"}`), &probe)
}
