package receptionist

import (
	"context"
	"time"

	"ai-receptionist/internal/agent/tools"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"
)

type storeAdapter struct{ db *store.DB }

func (s storeAdapter) InsertToolRun(convID, tool, input, output, errMsg string, latencyMS int64) error {
	return s.db.InsertToolRun(convID, tool, input, output, errMsg, latencyMS)
}

func (s storeAdapter) RecentMessages(convID string, limit int) ([]tools.Message, error) {
	msgs, err := s.db.RecentMessages(convID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]tools.Message, len(msgs))
	for i, m := range msgs {
		out[i] = tools.Message{Role: m.Role, Message: m.Message}
	}
	return out, nil
}

func (s storeAdapter) PauseContact(phone string, until time.Time) error {
	return s.db.PauseContact(phone, until)
}

func (s storeAdapter) GetOrCreateContact(phone string) (tools.Contact, error) {
	c, err := s.db.GetOrCreateContact(phone)
	if err != nil {
		return tools.Contact{}, err
	}
	return tools.Contact{Status: c.Status}, nil
}

func (s storeAdapter) RecentToolRuns(convID string, limit int) ([]tools.ToolRun, error) {
	runs, err := s.db.RecentToolRuns(convID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]tools.ToolRun, len(runs))
	for i, r := range runs {
		out[i] = tools.ToolRun{Tool: r.Tool, Input: r.Input, Output: r.Output, Error: r.Error}
	}
	return out, nil
}

type configAdapter struct{ cfg *config.Config }

func (c configAdapter) BusinessName() string        { return c.cfg.BusinessName }
func (c configAdapter) DisplayOwnerName() string  { return c.cfg.DisplayOwnerName() }
func (c configAdapter) OwnerNumber() string       { return c.cfg.OwnerNumber }
func (c configAdapter) PauseHours() int           { return c.cfg.PauseHours }

type waAdapter struct {
	wa  *whatsapp.Client
	cfg *config.Config
}

func (w waAdapter) SendOwnerAlert(ctx context.Context, ownerPhone, text string) error {
	jid := whatsapp.PhoneToJID(ownerPhone)
	return whatsapp.SendText(ctx, w.wa, jid, text)
}

func (h *Handler) toolRunContext(convID string) tools.RunContext {
	return tools.RunContext{
		ConvID: convID,
		Deps: tools.Deps{
			Store:    storeAdapter{db: h.store},
			Config:   configAdapter{cfg: h.cfg},
			WhatsApp: waAdapter{wa: h.wa, cfg: h.cfg},
			Calendar: h.calendar,
		},
	}
}
