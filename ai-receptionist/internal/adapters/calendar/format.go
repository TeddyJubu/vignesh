package calendar

import (
	"fmt"
	"time"
)

// formatHumanSlot renders a WhatsApp-friendly slot label (e.g. "Fri 3pm", "Mon 10:30am").
func formatHumanSlot(t time.Time) string {
	t = t.In(t.Location())
	h := t.Hour()
	m := t.Minute()
	period := "am"
	if h >= 12 {
		period = "pm"
	}
	hour12 := h % 12
	if hour12 == 0 {
		hour12 = 12
	}
	if m == 0 {
		return fmt.Sprintf("%s %d%s", t.Format("Mon"), hour12, period)
	}
	return fmt.Sprintf("%s %d:%02d%s", t.Format("Mon"), hour12, m, period)
}
