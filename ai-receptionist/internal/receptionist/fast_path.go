package receptionist

import "strings"

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
