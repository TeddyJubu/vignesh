package receptionist

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types"
)

// RunNudgeLoop sends one follow-up per stale collecting lead (receptionist mode).
func (h *Handler) RunNudgeLoop(ctx context.Context) {
	if !h.cfg.NudgeEnabled() {
		return
	}
	interval := time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	h.runNudgeTick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.runNudgeTick(ctx)
		}
	}
}

func (h *Handler) runNudgeTick(ctx context.Context) {
	if !h.cfg.LeadTrackingEnabled() || h.cfg.IsPersonal() {
		return
	}
	now := time.Now()
	if h.cfg.QuietHours.InQuietHours(now) {
		return
	}
	idle := time.Duration(h.cfg.NudgeIdleHours()) * time.Hour
	cutoff := time.Now().Add(-idle)
	phones, err := h.store.ListStaleCollecting(cutoff)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nudge list:", err)
		return
	}
	msg := h.cfg.NudgeMessage()
	for _, phone := range phones {
		if err := h.sendNudge(ctx, phone, msg); err != nil {
			fmt.Fprintln(os.Stderr, "nudge", phone, err)
			continue
		}
		_ = h.store.MarkNudgeSent(phone)
		fmt.Printf("nudge sent conv=%s\n", phone)
	}
}

func (h *Handler) sendNudge(ctx context.Context, convID, msg string) error {
	jid, err := convIDToJID(convID)
	if err != nil {
		return err
	}
	lock := h.chatLock(convID)
	lock.Lock()
	defer lock.Unlock()

	contact, err := h.store.GetContact(convID)
	if err != nil {
		return err
	}
	if contact.IsPaused(time.Now()) || contact.Status != "collecting" {
		return nil
	}
	if err := h.store.InsertMessage(convID, "assistant", msg); err != nil {
		return err
	}
	if err := whatsapp.SendText(ctx, h.wa, jid, msg); err != nil {
		return err
	}
	return h.store.TouchLastBotReply(convID)
}

func convIDToJID(convID string) (types.JID, error) {
	if strings.HasPrefix(convID, "self:") {
		return whatsapp.PhoneToJID(strings.TrimPrefix(convID, "self:")), nil
	}
	if strings.Contains(convID, "@") {
		return types.ParseJID(convID)
	}
	return whatsapp.PhoneToJID(convID), nil
}
