package receptionist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/lead"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const historyLimit = 10

type Handler struct {
	cfg       *config.Config
	store     *store.DB
	ai        *ai.Client
	wa        *whatsapp.Client
	promptTpl string

	mu       sync.Mutex
	chatMu   map[string]*sync.Mutex
}

func New(cfg *config.Config, db *store.DB, aiClient *ai.Client, wa *whatsapp.Client, promptTpl string) *Handler {
	return &Handler{
		cfg:       cfg,
		store:     db,
		ai:        aiClient,
		wa:        wa,
		promptTpl: promptTpl,
		chatMu:    make(map[string]*sync.Mutex),
	}
}

func (h *Handler) HandleMessage(ctx context.Context, v *events.Message) {
	own := h.wa.WM.Store.ID
	var ownJID types.JID
	if own != nil {
		ownJID = *own
	}
	in, ok := whatsapp.ShouldProcessInbound(v, whatsapp.InboundFilter{
		OwnerPhone:    h.cfg.OwnerNumber,
		ReplyToGroups: h.cfg.ReplyToGroups,
		ReplyToSelf:   h.cfg.SelfChatEnabled(),
		OwnJID:        ownJID,
		Sent:          h.wa.Sent,
		Normalize:     config.NormalizePhone,
	})
	if !ok {
		if os.Getenv("DEBUG_INBOUND") == "1" && v != nil {
			fmt.Fprintf(os.Stderr, "skip inbound chat=%s sender=%s fromMe=%v\n",
				v.Info.Chat, v.Info.Sender, v.Info.IsFromMe)
		}
		return
	}
	fmt.Printf("inbound conv=%s chat=%s text=%q\n", in.ConvID, v.Info.Chat, in.Text)

	lock := h.chatLock(in.ConvID)
	lock.Lock()
	defer lock.Unlock()

	if err := h.process(ctx, v, in); err != nil {
		fmt.Fprintln(os.Stderr, "receptionist:", in.ConvID, err)
		h.sendFailureReply(ctx, v, err)
	}
}

func (h *Handler) sendFailureReply(ctx context.Context, v *events.Message, err error) {
	msg := "I couldn't reach the AI right now — check the bot terminal (OPENAI_API_KEY or quota)."
	if strings.Contains(err.Error(), "403") || strings.Contains(strings.ToLower(err.Error()), "limit") {
		msg = "OpenAI quota or billing issue — check platform.openai.com and your API key."
	}
	_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, msg)
}

func (h *Handler) chatLock(phone string) *sync.Mutex {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.chatMu[phone]; ok {
		return m
	}
	m := &sync.Mutex{}
	h.chatMu[phone] = m
	return m
}

func (h *Handler) process(ctx context.Context, v *events.Message, in whatsapp.InboundContext) error {
	convID := in.ConvID
	text := in.Text

	contact, err := h.store.GetOrCreateContact(convID)
	if err != nil {
		return err
	}

	if err := h.store.InsertMessage(convID, "user", text); err != nil {
		return err
	}

	history, err := h.store.RecentMessages(convID, historyLimit)
	if err != nil {
		return err
	}

	leadData := contact.LeadData
	system := h.buildSystemPrompt(leadData, in)

	var histMsgs []ai.ChatMessage
	for _, m := range history {
		if m.Role == "user" || m.Role == "assistant" {
			histMsgs = append(histMsgs, ai.ChatMessage{Role: m.Role, Content: m.Message})
		}
	}
	if len(histMsgs) > 0 && histMsgs[len(histMsgs)-1].Role == "user" && histMsgs[len(histMsgs)-1].Content == text {
		histMsgs = histMsgs[:len(histMsgs)-1]
	}

	msgs := ai.BuildMessages(system, histMsgs, text)

	var reply string
	qualified := false
	var leadDataOut map[string]string
	var summary string

	if h.cfg.IsPersonal() {
		raw, err := h.ai.Complete(ctx, msgs, false)
		if err != nil {
			return err
		}
		reply = SanitizeReply(strings.TrimSpace(raw))
	} else {
		raw, err := h.ai.Complete(ctx, msgs, true)
		if err != nil {
			return err
		}
		parsed, err := ai.ParseStructuredResponse(raw)
		if err != nil {
			return err
		}
		reply = SanitizeReply(parsed.Reply)
		leadDataOut = lead.Merge(leadData, parsed.LeadUpdates)
		qualified = lead.IsQualified(leadDataOut)
		if parsed.Qualified {
			qualified = true
		}
		summary = parsed.Summary
	}

	status := contact.Status
	if h.cfg.LeadTrackingEnabled() {
		leadData = leadDataOut
		if status == "new" {
			status = "collecting"
		}
		if qualified {
			status = "qualified"
		}
		name := lead.DenormalizedName(leadData)
		leadJSON, _ := json.Marshal(leadData)
		if err := h.store.UpdateContact(convID, name, string(leadJSON), status); err != nil {
			return err
		}
	}

	if err := h.store.InsertMessage(convID, "assistant", reply); err != nil {
		return err
	}

	if err := whatsapp.SendText(ctx, h.wa, v.Info.Chat, reply); err != nil {
		return fmt.Errorf("send reply: %w", err)
	}

	if h.cfg.OwnerAlertsEnabled() && qualified && contact.Status != "notified" {
		if strings.TrimSpace(summary) == "" {
			summary = "Qualified lead via WhatsApp receptionist."
		}
		alert := lead.AdminSummary(h.cfg.BusinessName, in.Sender, leadData, summary)
		ownerJID := whatsapp.PhoneToJID(h.cfg.OwnerNumber)
		if err := whatsapp.SendText(ctx, h.wa, ownerJID, alert); err != nil {
			return fmt.Errorf("owner alert: %w", err)
		}
		name := lead.DenormalizedName(leadData)
		leadJSON, _ := json.Marshal(leadData)
		if err := h.store.UpdateContact(convID, name, string(leadJSON), "notified"); err != nil {
			return err
		}
		fmt.Println("Owner alerted for lead:", in.Sender)
	}

	return nil
}

func (h *Handler) buildSystemPrompt(leadData map[string]string, in whatsapp.InboundContext) string {
	p := h.promptTpl
	p = strings.ReplaceAll(p, "{{business_name}}", h.cfg.BusinessName)
	p = strings.ReplaceAll(p, "{{business_description}}", h.cfg.BusinessDescription)
	p = strings.ReplaceAll(p, "{{your_name}}", h.cfg.BusinessName)

	var b strings.Builder
	b.WriteString(p)
	if in.IsGroup {
		b.WriteString("\n\n## Chat context\nThis message is from a WhatsApp group. Reply in the thread; keep messages short. Sender: ")
		b.WriteString(in.SenderName)
		b.WriteString(" (")
		b.WriteString(in.Sender)
		b.WriteString(")\n")
	}
	if h.cfg.LeadTrackingEnabled() {
		missing := lead.Missing(leadData)
		leadJSON, _ := json.Marshal(leadData)
		b.WriteString("\n\n## Runtime context\n")
		b.WriteString("missing_fields: ")
		missJSON, _ := json.Marshal(missing)
		b.Write(missJSON)
		b.WriteString("\ncurrent_lead_data: ")
		b.Write(leadJSON)
		b.WriteString("\n")
	}
	return b.String()
}
