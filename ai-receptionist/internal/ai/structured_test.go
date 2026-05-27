package ai

import "testing"

func TestDecodeStructured_StripsFences(t *testing.T) {
	raw := "```json\n{\"reply\":\"hi\",\"lead_updates\":{},\"qualified\":false,\"summary\":\"\"}\n```"
	r, err := DecodeStructured(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Reply != "hi" {
		t.Fatalf("reply=%q", r.Reply)
	}
}

func TestStripCodeFences(t *testing.T) {
	if got := StripCodeFences("```\n{}\n```"); got != "{}" {
		t.Fatalf("got %q", got)
	}
}
