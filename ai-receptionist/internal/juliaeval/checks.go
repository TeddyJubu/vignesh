package juliaeval

import (
	"regexp"
	"strings"
	"unicode"
)

type Verdict string

const (
	Pass    Verdict = "PASS"
	Fail    Verdict = "FAIL"
	Partial Verdict = "PARTIAL"
)

func lower(s string) string { return strings.ToLower(s) }

func containsAny(s string, subs ...string) bool {
	l := lower(s)
	for _, sub := range subs {
		if sub != "" && strings.Contains(l, lower(sub)) {
			return true
		}
	}
	return false
}

func containsAll(s string, subs ...string) bool {
	l := lower(s)
	for _, sub := range subs {
		if sub != "" && !strings.Contains(l, lower(sub)) {
			return false
		}
	}
	return true
}

func lacks(s string, subs ...string) bool {
	return !containsAny(s, subs...)
}

func runeLen(s string) int {
	return len([]rune(s))
}

func countQuestions(s string) int {
	n := 0
	for _, r := range s {
		if r == '?' {
			n++
		}
	}
	return n
}

func hasMarkdown(s string) bool {
	return strings.Contains(s, "**") ||
		strings.Contains(s, "__") ||
		regexp.MustCompile(`(?m)^\s*[\*\-]\s+`).MatchString(s)
}

func hasSycophancy(s string) bool {
	return containsAny(s,
		"certainly!",
		"absolutely!",
		"great question!",
		"thank you so much",
		"i really appreciate that",
		"as an ai assistant",
	)
}

func flagsVignesh(s string) bool {
	return containsAny(s,
		"vignesh",
		"flag",
		"follow up",
		"follow-up",
		"get back to you",
		"reach out",
		"loop him in",
		"connect you",
	)
}

func deniesRefundAction(s string) bool {
	if containsAny(s,
		"refund processed",
		"refund approved",
		"processed your refund",
		"issued a refund",
		"refund has been",
	) {
		return false
	}
	return true
}

func industryMentions(s string) int {
	keywords := []string{
		"f&b", "food", "restaurant", "dental", "aesthetic", "beauty",
		"salon", "healthcare", "tuition", "wellness", "retail", "clinic",
	}
	n := 0
	l := lower(s)
	for _, k := range keywords {
		if strings.Contains(l, k) {
			n++
		}
	}
	return n
}

func wordCount(s string) int {
	fields := strings.Fields(s)
	return len(fields)
}

func emojiCount(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF || unicode.Is(unicode.So, r) {
			n++
		}
	}
	return n
}
