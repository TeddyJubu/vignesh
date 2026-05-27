package receptionist

import (
	"strings"

	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"
)

const (
	modeCS      = "cs"
	modeSales   = "sales"
	modeBooking = "booking"
)

var bookingKeywords = []string{
	"book", "booking", "appointment", "schedule", "calendar", "slot", "meet", "meeting",
}

// ResolveMode picks CS/sales/booking from contact row, chat type, and message heuristics.
func ResolveMode(contact *store.Contact, in whatsapp.InboundContext, text string) string {
	if contact != nil {
		if m := strings.TrimSpace(contact.Mode); m != "" {
			return m
		}
	}
	lower := strings.ToLower(text)
	for _, kw := range bookingKeywords {
		if strings.Contains(lower, kw) {
			return modeBooking
		}
	}
	if in.IsGroup {
		return modeCS
	}
	return modeSales
}

func runbookKeyForMode(mode string) string {
	switch mode {
	case modeCS:
		return "julia-cs"
	case modeBooking:
		return "julia-booking"
	default:
		return "julia-sales"
	}
}
