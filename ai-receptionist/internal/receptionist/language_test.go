package receptionist

import "testing"

func TestDetectLanguage(t *testing.T) {
	if DetectLanguage("Hello, I need a website") != "en" {
		t.Fatal("expected en")
	}
	if DetectLanguage("আমি একটি ওয়েবসাইট চাই") != "bn" {
		t.Fatal("expected bn")
	}
}
