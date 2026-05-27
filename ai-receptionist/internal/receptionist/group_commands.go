package receptionist

import (
	"context"
	"fmt"
	"strings"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

// IsGroupAdminKeyword detects owner-only group management commands.
func IsGroupAdminKeyword(text string) bool {
	t := strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(t, "create group ") ||
		strings.HasPrefix(t, "add to group ") ||
		strings.HasPrefix(t, "group invite ")
}

func (h *Handler) handleGroupAdmin(ctx context.Context, v *events.Message, in whatsapp.InboundContext) {
	if !h.cfg.Capabilities.GroupAdmin {
		_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Group admin is disabled in config.")
		return
	}
	lock := h.chatLock(in.ConvID)
	lock.Lock()
	defer lock.Unlock()

	text := strings.TrimSpace(in.Text)
	lower := strings.ToLower(text)

	switch {
	case strings.HasPrefix(lower, "create group "):
		rest := strings.TrimSpace(text[len("create group "):])
		name, phones := parseGroupCreateArgs(rest)
		if name == "" {
			_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Usage: create group <name> with <phone1>,<phone2>")
			return
		}
		info, err := whatsapp.CreateGroup(ctx, h.wa, name, phones)
		if err != nil {
			_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Could not create group: "+err.Error())
			return
		}
		link, _ := whatsapp.GetGroupInviteLink(ctx, h.wa, info.JID.String())
		msg := fmt.Sprintf("Created group %q (%s).", info.Name, info.JID.String())
		if link != "" {
			msg += "\nInvite: " + link
		}
		_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, msg)

	case strings.HasPrefix(lower, "add to group "):
		rest := strings.TrimSpace(text[len("add to group "):])
		groupJID, phones := parseGroupAddArgs(rest)
		if groupJID == "" || len(phones) == 0 {
			_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Usage: add to group <group_jid> <phone1>,<phone2>")
			return
		}
		if err := whatsapp.AddGroupParticipants(ctx, h.wa, groupJID, phones); err != nil {
			_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Could not add participants: "+err.Error())
			return
		}
		_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Added participants to "+groupJID)

	case strings.HasPrefix(lower, "group invite "):
		groupJID := strings.TrimSpace(text[len("group invite "):])
		link, err := whatsapp.GetGroupInviteLink(ctx, h.wa, groupJID)
		if err != nil {
			_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Could not get invite link: "+err.Error())
			return
		}
		_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Invite link: "+link)
	}
}

func parseGroupCreateArgs(rest string) (name string, phones []string) {
	lower := strings.ToLower(rest)
	if idx := strings.Index(lower, " with "); idx >= 0 {
		name = strings.TrimSpace(rest[:idx])
		phones = splitPhones(rest[idx+len(" with "):])
		return name, phones
	}
	name = strings.TrimSpace(rest)
	return name, nil
}

func parseGroupAddArgs(rest string) (groupJID string, phones []string) {
	parts := strings.Fields(rest)
	if len(parts) < 2 {
		return "", nil
	}
	groupJID = parts[0]
	phones = splitPhones(strings.Join(parts[1:], " "))
	return groupJID, phones
}

func splitPhones(s string) []string {
	s = strings.ReplaceAll(s, ";", ",")
	var out []string
	for _, p := range strings.Split(s, ",") {
		if n := config.NormalizePhone(strings.TrimSpace(p)); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// IsBookingCoordinationKeyword starts third-party booking flow (owner only).
func IsBookingCoordinationKeyword(text string) bool {
	t := strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(t, "book with ")
}

func (h *Handler) handleBookingCoordination(ctx context.Context, v *events.Message, in whatsapp.InboundContext) {
	if !h.cfg.Capabilities.OutboundBooking {
		return
	}
	guest := config.NormalizePhone(strings.TrimSpace(strings.TrimPrefix(strings.ToLower(in.Text), "book with ")))
	if guest == "" {
		_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Usage: book with <phone>")
		return
	}
	req := store.BookingRequest{
		OwnerConv:  in.ConvID,
		GuestPhone: guest,
		Status:     "awaiting_guest",
	}
	if err := h.store.InsertBookingRequest(req); err != nil {
		_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Could not start booking request.")
		return
	}
	guestJID := whatsapp.PhoneToJID(guest)
	msg := "Hi — Vignesh asked me to help schedule a call. What days/times work for you this week? (e.g. Tue 3pm SGT)"
	_ = whatsapp.SendText(ctx, h.wa, guestJID, msg)
	_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, fmt.Sprintf("Started booking request %s with %s.", req.ID, guest))
}
