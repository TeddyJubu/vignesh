package lead

import (
	"strings"
)

// Score returns hot, warm, or cold from collected lead fields (rule-based).
func Score(data map[string]string) string {
	budget := strings.ToLower(strings.TrimSpace(data["budget"]))
	timeline := strings.ToLower(strings.TrimSpace(data["timeline"]))
	service := strings.TrimSpace(data["service_needed"])

	if isColdSignal(budget, timeline, service) {
		return "cold"
	}
	if isHotSignal(budget, timeline, service) {
		return "hot"
	}
	return "warm"
}

func isColdSignal(budget, timeline, service string) bool {
	coldBudget := []string{"no budget", "not sure", "don't know", "dont know", "tbd", "n/a", "none", "free", "cheap only"}
	for _, p := range coldBudget {
		if strings.Contains(budget, p) {
			return true
		}
	}
	coldTimeline := []string{"not sure", "no rush", "just looking", "exploring", "someday", "later", "don't know", "dont know"}
	for _, p := range coldTimeline {
		if strings.Contains(timeline, p) {
			return true
		}
	}
	if service == "" {
		return true
	}
	return false
}

func isHotSignal(budget, timeline, service string) bool {
	if service == "" {
		return false
	}
	hotTimeline := []string{"asap", "urgent", "this week", "next week", "soon", "immediately", "1 week", "2 week", "days"}
	timelineHot := false
	for _, p := range hotTimeline {
		if strings.Contains(timeline, p) {
			timelineHot = true
			break
		}
	}
	if !timelineHot {
		return false
	}
	// Budget present and not explicitly low/unknown
	if budget == "" {
		return false
	}
	lowBudget := []string{"no budget", "not sure", "don't know", "dont know", "can't afford", "cant afford", "minimal"}
	for _, p := range lowBudget {
		if strings.Contains(budget, p) {
			return false
		}
	}
	return true
}
