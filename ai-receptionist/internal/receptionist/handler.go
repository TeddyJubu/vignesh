package receptionist

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/lead"
	"ai-receptionist/internal/ops"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/webhook"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const historyLimit = 10

type Handler struct {
	cfg          *config.Config
	store        *store.DB
	ai           *ai.Client
	wa           *whatsapp.Client
	promptTpl    string
	styleExtra   string
	debouncer    *Debouncer

	mu     sync.Mutex
	chatMu map[string]*sync.Mutex
}

func New(cfg *config.Config, db *store.DB, aiClient *ai.Client, wa *whatsapp.Client, promptTpl, styleExtra string) *Handler {
	h := &Handler{
		cfg:        cfg,
		store:      db,
		ai:         aiClient,
		wa:         wa,
		promptTpl:  promptTpl,
		styleExtra: styleExtra,
		chatMu:     make(map[string]*sync.Mutex),
	}
	h.debouncer = NewDebouncer(cfg.DebounceSeconds, h.handleDebounced)
	return h
}

func (h *Handler) HandleMessage(ctx context.Context, v *events.Message) {
	own := h.wa.WM.Store.ID
	var ownJID types.JID
	if own != nil {
		ownJID = *own
	}
	in, ok := whatsapp.ShouldProcessInbound(v, whatsapp.InboundFilter{
		OwnerPhone:     h.cfg.OwnerNumber,
		ReplyToGroups:  h.cfg.ReplyToGroups,
		ReplyToSelf:    h.cfg.SelfChatEnabled(),
		OwnJID:         ownJID,
		Sent:           h.wa.Sent,
		Normalize:      config.NormalizePhone,
		AllowedNumbers: h.cfg.AllowedNumbers,
		BlockedNumbers: h.cfg.BlockedNumbers,
	})
	if !ok {
		if os.Getenv("DEBUG_INBOUND") == "1" && v != nil {
			fmt.Fprintf(os.Stderr, "skip inbound chat=%s sender=%s fromMe=%v\n",
				v.Info.Chat, v.Info.Sender, v.Info.IsFromMe)
		}
		return
	}
	fmt.Printf("inbound conv=%s chat=%s text=%q\n", in.ConvID, v.Info.Chat, in.Text)

	if IsPauseKeyword(in.Text) {
		if canPauseSender(in, h.cfg.OwnerNumber) {
			h.handlePause(ctx, v, in)
		}
		return
	}

	h.debouncer.Enqueue(ctx, v, in)
}

func (h *Handler) handlePause(ctx context.Context, v *events.Message, in whatsapp.InboundContext) {
	lock := h.chatLock(in.ConvID)
	lock.Lock()
	defer lock.Unlock()

	h.debouncer.Cancel(in.ConvID)

	until := time.Now().Add(time.Duration(h.cfg.PauseHours) * time.Hour)
	if _, err := h.store.GetOrCreateContact(in.ConvID); err != nil {
		fmt.Fprintln(os.Stderr, "pause:", err)
		return
	}
	if err := h.store.PauseContact(in.ConvID, until); err != nil {
		fmt.Fprintln(os.Stderr, "pause:", err)
		return
	}
	ack := "Got it — I'll stay quiet in this chat until you message again or the pause expires."
	_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, ack)
	fmt.Printf("paused conv=%s until %s\n", in.ConvID, until.Format(time.RFC3339))
}

func (h *Handler) handleDebounced(ctx context.Context, v *events.Message, in whatsapp.InboundContext, combinedText string) {
	lock := h.chatLock(in.ConvID)
	lock.Lock()
	defer lock.Unlock()

	if err := h.process(ctx, v, in, combinedText); err != nil {
		fmt.Fprintln(os.Stderr, "receptionist:", in.ConvID, err)
		ops.AppendErrorLog("receptionist/"+in.ConvID, err)
		h.sendFailureReply(ctx, v, err)
	}
}

func (h *Handler) sendFailureReply(ctx context.Context, v *events.Message, err error) {
	msg := "I couldn't reach the AI right now — check the bot terminal (OLLAMA_API_KEY or Ollama Cloud status)."
	if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "401") {
		msg = "Ollama Cloud auth failed — check OLLAMA_API_KEY at ollama.com/settings/keys."
	}
	if strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "limit") {
		msg = "Ollama rate limit — try again shortly."
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

func (h *Handler) process(ctx context.Context, v *events.Message, in whatsapp.InboundContext, text string) error {
	convID := in.ConvID

	_ = h.store.ClearPauseIfExpired(convID, time.Now())
	contact, err := h.store.GetOrCreateContact(convID)
	if err != nil {
		return err
	}
	if contact.IsPaused(time.Now()) {
		if err := h.store.InsertMessage(convID, "user", text); err != nil {
			return err
		}
		fmt.Printf("skip AI (paused) conv=%s\n", convID)
		return nil
	}

	if contact.Language == "" {
		lang := DetectLanguage(text)
		if err := h.store.SetContactLanguage(convID, lang); err != nil {
			return err
		}
		contact.Language = lang
	}

	if h.cfg.QuietHours.InQuietHours(time.Now()) {
		if err := h.store.InsertMessage(convID, "user", text); err != nil {
			return err
		}
		reply := h.cfg.QuietHours.AutoReplyMessage()
		sendQuietReply := reply != "" && shouldSendQuietHoursReply(h.cfg.QuietHours, contact.LastBotReplyAt, time.Now())
		if sendQuietReply {
			if err := h.store.InsertMessage(convID, "assistant", reply); err != nil {
				return err
			}
			if err := whatsapp.SendText(ctx, h.wa, v.Info.Chat, reply); err != nil {
				return err
			}
			_ = h.store.TouchLastBotReply(convID)
		}
		fmt.Printf("quiet hours skip AI conv=%s\n", convID)
		return nil
	}

	if err := h.store.InsertMessage(convID, "user", text); err != nil {
		return err
	}

	history, err := h.store.RecentMessages(convID, historyLimit)
	if err != nil {
		return err
	}

	leadData := contact.LeadData
	system := h.buildSystemPrompt(leadData, in, contact.Language)

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

	status := mergeLeadStatus(contact.Status, qualified, h.cfg.LeadTrackingEnabled())
	if h.cfg.LeadTrackingEnabled() {
		leadData = leadDataOut
		name := lead.DenormalizedName(leadData)
		leadJSON, _ := json.Marshal(leadData)
		score := ""
		if qualified {
			score = lead.Score(leadData)
		}
		if err := h.store.UpdateContactWithScore(convID, name, string(leadJSON), status, score); err != nil {
			return err
		}
	}

	if err := h.store.InsertMessage(convID, "assistant", reply); err != nil {
		return err
	}

	contact, err = h.store.GetContact(convID)
	if err != nil {
		return err
	}
	if contact.IsPaused(time.Now()) {
		fmt.Printf("skip send (paused after AI) conv=%s\n", convID)
		return nil
	}

	whatsapp.SendTyping(ctx, h.wa, v.Info.Chat, true)
	if err := whatsapp.SendText(ctx, h.wa, v.Info.Chat, reply); err != nil {
		return fmt.Errorf("send reply: %w", err)
	}
	_ = h.store.TouchLastBotReply(convID)

	if h.cfg.LeadTrackingEnabled() && qualified {
		if err := h.notifyQualifiedLead(ctx, convID, in, leadData, summary, contact); err != nil {
			return err
		}
	}

	return nil
}

// mergeLeadStatus updates funnel status without downgrading notified leads.
func mergeLeadStatus(current string, qualified, tracking bool) string {
	if !tracking {
		return current
	}
	if current == "notified" {
		return "notified"
	}
	if current == "new" {
		current = "collecting"
	}
	if qualified {
		return "qualified"
	}
	return current
}

func shouldSendQuietHoursReply(q config.QuietHours, lastBot *time.Time, now time.Time) bool {
	if lastBot == nil {
		return true
	}
	// One auto-reply per quiet-hours window; a reply sent during quiet hours suppresses repeats.
	return !q.InQuietHours(*lastBot)
}

func (h *Handler) notifyQualifiedLead(ctx context.Context, convID string, in whatsapp.InboundContext, leadData map[string]string, summary string, contact *store.Contact) error {
	if contact.Status == "notified" {
		return nil
	}
	if strings.TrimSpace(summary) == "" {
		summary = "Qualified lead via WhatsApp receptionist."
	}
	if h.cfg.WebhookURL != "" && contact.WebhookSentAt == nil {
		if err := webhook.NotifyQualify(ctx, h.cfg.WebhookURL, h.cfg.WebhookSecret,
			h.cfg.BusinessName, convID, in.Sender, leadData, summary); err != nil {
			ops.AppendErrorLog("webhook", err)
			fmt.Fprintln(os.Stderr, "webhook:", err)
		} else {
			_ = h.store.MarkWebhookSent(convID)
		}
	}
	if !h.cfg.OwnerAlertsEnabled() {
		name := lead.DenormalizedName(leadData)
		leadJSON, _ := json.Marshal(leadData)
		return h.store.UpdateContact(convID, name, string(leadJSON), "notified")
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
	return nil
}

func (h *Handler) buildSystemPrompt(leadData map[string]string, in whatsapp.InboundContext, language string) string {
	p := h.promptTpl
	p = strings.ReplaceAll(p, "{{business_name}}", h.cfg.BusinessName)
	p = strings.ReplaceAll(p, "{{business_description}}", h.cfg.BusinessDescription)
	p = strings.ReplaceAll(p, "{{your_name}}", h.cfg.BusinessName)

	var b strings.Builder
	b.WriteString(p)
	if h.styleExtra != "" {
		b.WriteString("\n\n## Style examples\n")
		b.WriteString(h.styleExtra)
		b.WriteString("\n")
	}
	if language != "" {
		b.WriteString("\n\n## Language\n")
		b.WriteString(languagePromptLine(language))
		b.WriteString("\n")
	}
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
