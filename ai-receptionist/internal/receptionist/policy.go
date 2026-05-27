package receptionist

import (
	"encoding/json"
	"strings"

	"ai-receptionist/internal/agent"
)

// ApplyBookingPolicy ensures replies do not confirm bookings without successful book tool.
func ApplyBookingPolicy(reply string, toolResults []agent.ToolResult) string {
	if !bookingConfirm.MatchString(reply) {
		return reply
	}
	for _, r := range toolResults {
		if strings.ToLower(r.Tool) != "book_appointment" || r.Error != "" {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(r.Output), &m) == nil {
			if booked, _ := m["booked"].(bool); booked {
				return reply
			}
		}
	}
	return deferReply
}
