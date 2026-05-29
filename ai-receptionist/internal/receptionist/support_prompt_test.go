package receptionist

import (
	"strings"
	"testing"

	"ai-receptionist/internal/store"
)

func TestBuildSupportUserTurn_structure(t *testing.T) {
	got := BuildSupportUserTurn("KB line", "user: hi\nassistant: hello", "What is GEO?")
	for _, want := range []string{
		"EPICWARE KNOWLEDGE BASE:",
		"KB line",
		"CONVERSATION HISTORY:",
		"user: hi",
		"CURRENT MESSAGE:",
		"What is GEO?",
		"TASK:",
		"flag it for Vignesh",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildBundledSupportMessages_excludesDuplicateCurrent(t *testing.T) {
	hist := []store.Message{
		{Role: "user", Message: "Hi"},
		{Role: "assistant", Message: "Hello"},
		{Role: "user", Message: "pricing?"},
	}
	msgs := BuildBundledSupportMessages("SOUL", "KB", hist, "pricing?")
	if len(msgs) != 2 {
		t.Fatalf("want 2 messages got %d", len(msgs))
	}
	if msgs[0].Role != "system" || !strings.Contains(msgs[0].Content, "SOUL") {
		t.Fatalf("bad system: %q", msgs[0].Content)
	}
	if strings.Count(msgs[1].Content, "pricing?") != 1 {
		t.Fatalf("current message should appear once in user turn: %q", msgs[1].Content)
	}
}
