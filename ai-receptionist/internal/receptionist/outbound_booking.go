package receptionist

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

// TryHandleGuestBookingReply routes guest slot confirmations back to active booking requests.
func (h *Handler) TryHandleGuestBookingReply(ctx context.Context, v *events.Message, in whatsapp.InboundContext) bool {
	if in.IsGroup {
		return false
	}
	phone := config.NormalizePhone(in.Sender)
	if phone == "" {
		return false
	}
	req, err := h.store.GetActiveBookingByGuest(phone)
	if err != nil || req == nil {
		return false
	}
	if req.Status != "awaiting_guest" && req.Status != "awaiting_guest_choice" {
		return false
	}

	text := strings.TrimSpace(strings.ToLower(in.Text))
	var slots []string
	_ = json.Unmarshal([]byte(req.GuestSlotsJSON), &slots)
	chosen := ""
	switch {
	case text == "1" || strings.Contains(text, "option 1") || (len(slots) > 0 && strings.Contains(text, strings.ToLower(slots[0]))):
		if len(slots) > 0 {
			chosen = slots[0]
		}
	case text == "2" || strings.Contains(text, "option 2") || (len(slots) > 1 && strings.Contains(text, strings.ToLower(slots[1]))):
		if len(slots) > 1 {
			chosen = slots[1]
		}
	case text == "3" || strings.Contains(text, "option 3") || (len(slots) > 2 && strings.Contains(text, strings.ToLower(slots[2]))):
		if len(slots) > 2 {
			chosen = slots[2]
		}
	default:
		if n, err := strconv.Atoi(text); err == nil && n >= 1 && n <= len(slots) {
			chosen = slots[n-1]
		}
	}
	if chosen == "" {
		chosen = strings.TrimSpace(in.Text)
	}

	_ = h.store.UpdateBookingRequestStatus(req.ID, "confirmed", chosen, "pending-calendar")
	guestJID := whatsapp.PhoneToJID(phone)
	_ = whatsapp.SendText(ctx, h.wa, guestJID, fmt.Sprintf("Confirmed — %s is booked with Vignesh. You'll get a calendar invite shortly.", chosen))
	if req.OwnerConv != "" {
		ownerChat := whatsapp.PhoneToJID(h.cfg.OwnerNumber)
		_ = whatsapp.SendText(ctx, h.wa, ownerChat, fmt.Sprintf("Booking confirmed with %s (%s) at %s.", req.GuestName, phone, chosen))
	}
	return true
}
