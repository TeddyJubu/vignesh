package receptionist

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ai-receptionist/internal/adapters/calendar"
	"ai-receptionist/internal/agent"
	"ai-receptionist/internal/agent/tools"
	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/lead"
	"ai-receptionist/internal/memory"
	"ai-receptionist/internal/ops"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/webhook"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const historyLimit = 10

type Handler struct {
	cfg            *config.Config
	store          *store.DB
	ai             ai.Provider
	wa             *whatsapp.Client
	promptTpl      string
	styleExtra     string
	instructionsMD string
	debouncer      *Debouncer
	promptBuilder  *PromptBuilder
	toolReg        *tools.Registry
	calendar       calendar.Calendar
	graphiti       *memory.Client

	chatLocks *convCache
	ackMu     sync.Mutex
}

func New(cfg *config.Config, db *store.DB, aiClient ai.Provider, wa *whatsapp.Client, promptTpl, styleExtra, instructionsMD string) *Handler {
	graphitiURL := strings.TrimSpace(os.Getenv("GRAPHITI_URL"))
	h := &Handler{
		cfg:            cfg,
		store:          db,
		ai:             aiClient,
		wa:             wa,
		promptTpl:      promptTpl,
		styleExtra:     styleExtra,
		instructionsMD: instructionsMD,
		promptBuilder:  NewPromptBuilder(cfg, db, instructionsMD),
		toolReg:        tools.DefaultRegistry(),
		calendar:       calendar.New(),
		graphiti:       memory.NewClient(graphitiURL),
		chatLocks:      newConvCache(),
	}
	agent.SetDefaultRegistry(h.toolReg)
	h.debouncer = NewDebouncer(cfg.DebounceSeconds, h.handleDebounced)
	return h
}

func (h *Handler) HandleMessage(ctx context.Context, v *events.Message) {
	own := h.wa.WM.Store.ID
	var ownJID types.JID
	if own != nil {
		ownJID = *own
	}
	replyGroups := h.cfg.ReplyToGroups && h.cfg.GroupCSAllowed()
	in, ok := whatsapp.ShouldProcessInbound(v, whatsapp.InboundFilter{
		OwnerPhone:          h.cfg.OwnerNumber,
		ReplyToGroups:       replyGroups,
		ReplyToSelf:         h.cfg.SelfChatEnabled(),
		OwnJID:              ownJID,
		Sent:                h.wa.Sent,
		Normalize:           config.NormalizePhone,
		AllowedNumbers:      h.cfg.AllowedNumbers,
		BlockedNumbers:      h.cfg.BlockedNumbers,
		SupportGroupJIDs:    h.cfg.SupportGroupJIDs,
		GroupReplyPolicy:    h.cfg.ResolvedGroupReplyPolicy(),
		GroupMentionAliases: h.cfg.GroupMentionAliases,
	})
	if !ok {
		if os.Getenv("DEBUG_INBOUND") == "1" && v != nil {
			fmt.Fprintf(os.Stderr, "skip inbound chat=%s sender=%s fromMe=%v\n",
				v.Info.Chat, v.Info.Sender, v.Info.IsFromMe)
		}
		return
	}
	fmt.Printf("inbound conv=%s chat=%s text=%q\n", in.ConvID, v.Info.Chat, in.Text)

	if IsResetKeyword(in.Text) {
		if canPauseSender(in, h.cfg.OwnerNumber) {
			h.handleReset(ctx, v, in)
		}
		return
	}
	if IsGroupAdminKeyword(in.Text) && canPauseSender(in, h.cfg.OwnerNumber) {
		h.handleGroupAdmin(ctx, v, in)
		return
	}
	if IsBookingCoordinationKeyword(in.Text) && canPauseSender(in, h.cfg.OwnerNumber) {
		h.handleBookingCoordination(ctx, v, in)
		return
	}
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

func (h *Handler) handleReset(ctx context.Context, v *events.Message, in whatsapp.InboundContext) {
	lock := h.chatLock(in.ConvID)
	lock.Lock()
	defer lock.Unlock()

	h.debouncer.Cancel(in.ConvID)

	if err := h.store.ResetConversation(in.ConvID); err != nil {
		fmt.Fprintln(os.Stderr, "reset:", err)
		_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "I couldn’t reset this chat right now — check the bot logs.")
		return
	}

	// Also reset the in-memory ack cooldown key (DB meta is cleared in ResetConversation).
	h.ackMu.Lock()
	h.chatLocks.Set("ack:"+in.ConvID, time.Time{})
	h.ackMu.Unlock()

	_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, "Session refreshed — starting fresh in this chat.")
	fmt.Printf("reset conv=%s\n", in.ConvID)
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
	provider := "AI"
	if h.ai != nil && h.ai.Name() != "" {
		provider = strings.ToUpper(h.ai.Name())
	}
	msg := "I couldn't reach the AI right now — check the bot terminal (provider auth/network)."
	if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "401") {
		if strings.Contains(strings.ToLower(provider), "openai") {
			msg = "OpenAI auth failed — check OPENAI_API_KEY."
		} else {
			msg = "Ollama Cloud auth failed — check OLLAMA_API_KEY at ollama.com/settings/keys."
		}
	}
	if strings.Contains(err.Error(), "429") || strings.Contains(strings.ToLower(err.Error()), "limit") {
		msg = provider + " rate limit — try again shortly."
	}
	_ = whatsapp.SendText(ctx, h.wa, v.Info.Chat, msg)
}

func (h *Handler) chatLock(phone string) *sync.Mutex {
	v := h.chatLocks.GetOrSet(phone, func() any { return &sync.Mutex{} })
	return v.(*sync.Mutex)
}

func (h *Handler) parseStructuredWithRepair(ctx context.Context, raw string) (*ai.StructuredResponse, error) {
	parsed, err := ai.DecodeStructured(raw)
	if err == nil {
		return parsed, nil
	}
	repairCtx, cancel := budgetCtx(ctx, 8*time.Second)
	defer cancel()
	repaired, err2 := h.ai.Complete(repairCtx, ai.RepairStructuredPrompt(raw), true)
	if err2 != nil {
		return nil, err
	}
	return ai.DecodeStructured(repaired)
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
	mode := ResolveMode(contact, in, text)
	if contact.Mode != mode {
		_ = h.store.SetContactMode(convID, mode)
		contact.Mode = mode
	}
	system, err := h.buildSystemPrompt(convID, leadData, in, contact.Language, mode)
	if err != nil {
		return err
	}

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

	overallCtx, cancel := context.WithTimeout(ctx, overallAITimeout)
	defer cancel()

	// Low perceived latency: start typing immediately. We'll stop typing right before returning.
	whatsapp.SetTyping(overallCtx, h.wa, v.Info.Chat, true)
	defer whatsapp.SetTyping(context.Background(), h.wa, v.Info.Chat, false)

	ackCtx, cancelAck := context.WithCancel(overallCtx)
	defer cancelAck()
	go h.maybeSendAck(ackCtx, v.Info.Chat, convID)

	providerName := "unknown"
	if h.ai != nil {
		providerName = h.ai.Name()
	}
	raw, structuredOut, intermediate, toolResults, err := h.completeWithPlanner(overallCtx, convID, msgs, !h.cfg.IsPersonal(), providerName)
	if err != nil {
		return err
	}
	if intermediate {
		cancelAck()
	}

	if h.cfg.IsPersonal() {
		reply = SanitizeReply(strings.TrimSpace(raw))
	} else {
		if !structuredOut {
			reply = SanitizeReply(strings.TrimSpace(raw))
			goto SEND_REPLY
		}
		parsed, err := h.parseStructuredWithRepair(overallCtx, raw)
		if err != nil {
			return err
		}
		reply = SanitizeReplyWithTools(parsed.Reply, toolResults)
		leadDataOut = lead.Merge(leadData, parsed.LeadUpdates)
		qualified = lead.IsQualified(leadDataOut)
		if parsed.Qualified {
			qualified = true
		}
		summary = parsed.Summary
	}

SEND_REPLY:
	canUpdateLead := !h.cfg.IsPersonal() && structuredOut
	status := mergeLeadStatus(contact.Status, qualified, h.cfg.LeadTrackingEnabled() && canUpdateLead)
	if h.cfg.LeadTrackingEnabled() && canUpdateLead {
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

	sendStart := time.Now()
	if err := whatsapp.SendText(ctx, h.wa, v.Info.Chat, reply); err != nil {
		logTurnPhase(convID, providerName, "send", sendStart, err)
		traceTurn(h.store, convID, "send", sendStart, err)
		return fmt.Errorf("send reply: %w", err)
	}
	logTurnPhase(convID, providerName, "send", sendStart, nil)
	traceTurn(h.store, convID, "send", sendStart, nil)
	_ = h.store.TouchLastBotReply(convID)
	if !intermediate {
		_ = h.store.ClearAgentState(convID)
	}

	if h.cfg.LeadTrackingEnabled() && canUpdateLead && qualified {
		if err := h.notifyQualifiedLead(ctx, convID, in, leadData, summary, contact); err != nil {
			return err
		}
	}

	return nil
}

// completeWithPlanner returns raw AI output, whether receptionist JSON mode applies,
// whether the reply is an intermediate planner question (not final collation), and any error.
func (h *Handler) completeWithPlanner(ctx context.Context, convID string, msgs []ai.ChatMessage, structured bool, provider string) (raw string, structuredOut bool, intermediate bool, toolResults []agent.ToolResult, err error) {
	// Resume if we have pending agent state.
	if st, err := h.store.GetAgentState(convID); err == nil && st != nil {
		var state agent.State
		if json.Unmarshal([]byte(st.StateJSON), &state) == nil && len(state.Plan.Questions) > 0 {
			// Consume the incoming user message as an answer to the next pending question.
			if state.Answers == nil {
				state.Answers = map[string]string{}
			}
			if state.NextQIndex < 0 {
				state.NextQIndex = 0
			}
			if state.NextQIndex < len(state.Plan.Questions) {
				// The actual answer text is always the last user message in msgs.
				if len(msgs) > 0 {
					last := msgs[len(msgs)-1]
					if strings.ToLower(last.Role) == "user" {
						q := state.Plan.Questions[state.NextQIndex]
						state.Answers[q] = last.Content
						state.NextQIndex++
					}
				}
			}
			if state.NextQIndex < len(state.Plan.Questions) {
				// Ask the next question and persist state.
				if err := h.store.UpsertAgentState(convID, state); err != nil {
					return "", false, true, nil, err
				}
				return state.Plan.Questions[state.NextQIndex], false, true, nil, nil
			}
			// Done collecting answers — run tools + collate (keep state until final send succeeds).
			out, results, err := h.runPlanAndCollate(ctx, convID, &state.Plan, state.Answers, structured, provider)
			return out, true, false, results, err
		}
	} else if err != nil && err != sql.ErrNoRows {
		return "", false, false, nil, err
	}

	planCtx, cancel := budgetCtx(ctx, 6*time.Second)
	defer cancel()
	planStart := time.Now()
	rawPlan, err := h.ai.Complete(planCtx, buildPlannerMessages(msgs, structured, h.toolReg), false)
	logTurnPhase(convID, provider, "planner", planStart, err)
	traceTurn(h.store, convID, "planner", planStart, err)
	if err != nil {
		fallbackCtx, cancelFB := budgetCtx(ctx, 20*time.Second)
		defer cancelFB()
		out, err := h.ai.Complete(fallbackCtx, msgs, structured)
		return out, true, false, nil, err
	}
	plan, err := agent.ParsePlan(rawPlan)
	if err != nil {
		repairCtx, cancelRepair := budgetCtx(ctx, 4*time.Second)
		repaired, errRepair := h.ai.Complete(repairCtx, buildPlannerRepairMessages(rawPlan, structured, h.toolReg), false)
		cancelRepair()
		if errRepair == nil {
			plan, err = agent.ParsePlan(repaired)
		}
	}
	if err != nil {
		fallbackCtx, cancelFB := budgetCtx(ctx, 20*time.Second)
		defer cancelFB()
		out, err := h.ai.Complete(fallbackCtx, msgs, structured)
		return out, true, false, nil, err
	}
	if err := h.toolReg.ValidatePlannerTools(planAgentTools(plan)); err != nil {
		fallbackCtx, cancelFB := budgetCtx(ctx, 20*time.Second)
		defer cancelFB()
		out, err := h.ai.Complete(fallbackCtx, msgs, structured)
		return out, true, false, nil, err
	}
	if len(plan.Questions) > 0 {
		state := agent.State{
			Plan:          *plan,
			NextQIndex:    0,
			Answers:       map[string]string{},
			StartedAtUNIX: time.Now().Unix(),
		}
		if err := h.store.UpsertAgentState(convID, state); err != nil {
			return "", false, true, nil, err
		}
		return plan.Questions[0], false, true, nil, nil
	}
	out, results, err := h.runPlanAndCollate(ctx, convID, plan, nil, structured, provider)
	return out, true, false, results, err
}

func (h *Handler) runPlanAndCollate(ctx context.Context, convID string, plan *agent.Plan, answers map[string]string, structured bool, provider string) (string, []agent.ToolResult, error) {
	rc := h.toolRunContext(convID)
	rc.Deps.Calendar = h.calendar
	toolCtx := tools.ContextWithCalendar(ctx, h.calendar)

	agentCtx, cancel := budgetCtx(toolCtx, 10*time.Second)
	defer cancel()
	toolStart := time.Now()
	results := agent.RunToolsParallel(agentCtx, rc, plan.Agents)
	logTurnPhase(convID, provider, "tools", toolStart, nil)
	traceTurn(h.store, convID, "tools", toolStart, nil)

	collateCtx, cancel2 := budgetCtx(ctx, 20*time.Second)
	defer cancel2()
	collateStart := time.Now()
	collateMsgs := buildCollationMessages(plan, answers, results, structured)
	out, err := h.ai.Complete(collateCtx, collateMsgs, structured)
	logTurnPhase(convID, provider, "collate", collateStart, err)
	traceTurn(h.store, convID, "collate", collateStart, err)
	return out, results, err
}

func (h *Handler) maybeSendAck(ctx context.Context, chat types.JID, convID string) {
	defer func() { recover() }()
	select {
	case <-ctx.Done():
		return
	case <-time.After(ackDelay):
	}
	if ctx.Err() != nil {
		return
	}
	if !h.shouldSendAck(convID) {
		return
	}
	if err := whatsapp.SendText(ctx, h.wa, chat, "Got it — checking now."); err == nil {
		h.markAckSent(convID)
	}
}

func (h *Handler) shouldSendAck(convID string) bool {
	now := time.Now()
	if t, err := h.store.GetLastAckAt(convID); err == nil && t != nil && now.Sub(*t) < ackCooldown {
		return false
	}
	h.ackMu.Lock()
	defer h.ackMu.Unlock()
	if v, ok := h.chatLocks.Get("ack:" + convID); ok {
		if t, ok := v.(time.Time); ok && now.Sub(t) < ackCooldown {
			return false
		}
	}
	return true
}

func (h *Handler) markAckSent(convID string) {
	now := time.Now()
	_ = h.store.TouchLastAckAt(convID, now)
	h.ackMu.Lock()
	h.chatLocks.Set("ack:"+convID, now)
	h.ackMu.Unlock()
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

func planAgentTools(plan *agent.Plan) []string {
	if plan == nil {
		return nil
	}
	out := make([]string, 0, len(plan.Agents))
	for _, a := range plan.Agents {
		if t := strings.TrimSpace(a.Tool); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (h *Handler) buildSystemPrompt(convID string, leadData map[string]string, in whatsapp.InboundContext, language, mode string) (string, error) {
	stack, err := h.promptBuilder.Build(convID, mode)
	if err != nil {
		return "", err
	}

	p := h.promptTpl
	p = strings.ReplaceAll(p, "{{business_name}}", h.cfg.BusinessName)
	p = strings.ReplaceAll(p, "{{business_description}}", h.cfg.BusinessDescription)
	p = strings.ReplaceAll(p, "{{your_name}}", h.cfg.DisplayOwnerName())

	var b strings.Builder
	b.WriteString(stack)
	b.WriteString("\n\n---\n\n")
	b.WriteString(p)
	if h.styleExtra != "" {
		b.WriteString("\n\n## Style examples\n")
		b.WriteString(h.styleExtra)
		b.WriteString("\n")
	}
	if os.Getenv("MEMORY_RECALL_IN_PROMPT") == "1" && h.graphiti != nil && h.graphiti.Enabled() {
		recallCtx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
		defer cancel()
		if rr, err := h.graphiti.Recall(recallCtx, convID, "", 5); err == nil && rr != nil && strings.TrimSpace(rr.Snippet) != "" {
			b.WriteString("\n\n## Recall (memory)\n")
			// Bound recall snippet size defensively (service also bounds).
			snip := strings.TrimSpace(rr.Snippet)
			if len(snip) > 1800 {
				snip = snip[:1800]
			}
			b.WriteString(snip)
			b.WriteString("\n")
		}
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
	return b.String(), nil
}
