package receptionist

import (
	"regexp"
	"strings"
)

var schedulingTimeRe = regexp.MustCompile(`(?i)(?:\d{1,2}\s*(?:am|pm)|\d{1,2}:\d{2})`)

var schedulingDayNames = []string{
	"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
	"mon", "tue", "wed", "thu", "fri", "sat", "sun",
}

// looksLikeSchedulingUpdate returns true for messages that appear to specify a slot or time.
func looksLikeSchedulingUpdate(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, d := range schedulingDayNames {
		if strings.Contains(lower, d) {
			return true
		}
	}
	if schedulingTimeRe.MatchString(text) {
		return true
	}
	for _, kw := range []string{"sgt", "singapore", "timezone"} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// needsPlannerPath returns true when multi-step planner/tools/collation should run.
func needsPlannerPath(mode, userText string, hasPendingPlan bool) bool {
	if hasPendingPlan {
		return true
	}
	if mode == modeBooking {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(userText))
	for _, kw := range plannerPathKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// shouldRecallMemory limits Graphiti recall to turns that likely benefit from memory.
func shouldRecallMemory(mode, userText string) bool {
	if mode == modeBooking {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(userText))
	for _, kw := range plannerPathKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

var plannerPathKeywords = []string{
	"calendar", "availability", "available",
	"book", "booking", "appointment", "schedule", "slot", "meet", "meeting",
	"email", "escalate", "human", "owner", "vignesh",
	"research", "scrape", "webhook", "csv",
	"price", "pricing", "quote", "cost", "budget",
	"service", "services", "website",
}
