package config

import (
	"fmt"
	"strings"
	"time"
)

// InQuietHours reports whether now falls outside the configured active window.
// Supports windows that span midnight (e.g. 22:00–08:00).
func (q QuietHours) InQuietHours(now time.Time) bool {
	if !q.Enabled {
		return false
	}
	loc, err := time.LoadLocation(strings.TrimSpace(q.TZ))
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	startMin, ok1 := parseHM(q.Start)
	endMin, ok2 := parseHM(q.End)
	if !ok1 || !ok2 {
		return false
	}
	cur := local.Hour()*60 + local.Minute()
	if startMin <= endMin {
		return cur < startMin || cur >= endMin
	}
	// overnight: quiet from start through midnight and from midnight until end
	return cur >= startMin || cur < endMin
}

func (q QuietHours) AutoReplyMessage() string {
	if m := strings.TrimSpace(q.Message); m != "" {
		return m
	}
	return "Thanks for your message — we're offline right now and will reply during business hours."
}

func parseHM(s string) (minutes int, ok bool) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, false
	}
	h, m := 0, 0
	if _, err := fmt.Sscanf(parts[0], "%d", &h); err != nil {
		return 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &m); err != nil {
		return 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
