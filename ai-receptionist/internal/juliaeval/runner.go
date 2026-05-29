package juliaeval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/lead"
	"ai-receptionist/internal/models"
	"ai-receptionist/internal/receptionist"
	"ai-receptionist/internal/settings"
	"ai-receptionist/internal/store"
)

// Runner drives live model evals using the same prompt stack as production.
type Runner struct {
	Cfg           *config.Config
	DB            *store.DB
	AI            ai.Provider
	PromptTpl     string
	StyleExtra    string
	Instructions  string
	PromptBuilder *receptionist.PromptBuilder
}

func NewRunner(cfg *config.Config, db *store.DB, promptTpl, styleExtra, instructionsMD string) (*Runner, error) {
	resolver := settings.New(db)
	models.SetConfigModel(cfg.Model)
	aiClient, err := ai.NewProviderForModel(cfg, resolver, models.GetModel(""))
	if err != nil {
		return nil, err
	}
	return &Runner{
		Cfg:           cfg,
		DB:            db,
		AI:            aiClient,
		PromptTpl:     promptTpl,
		StyleExtra:    styleExtra,
		Instructions:  instructionsMD,
		PromptBuilder: receptionist.NewPromptBuilder(cfg, db, instructionsMD),
	}, nil
}

type CaseResult struct {
	Case    Case
	Replies []string
	Verdict Verdict
	Note    string
}

func (r *Runner) RunAll(ctx context.Context) ([]CaseResult, error) {
	var out []CaseResult
	for _, tc := range AllCases() {
		res, err := r.runCase(ctx, tc)
		if err != nil {
			return out, fmt.Errorf("%s: %w", tc.ID, err)
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *Runner) runCase(ctx context.Context, tc Case) (CaseResult, error) {
	convID := "juliaeval_" + strings.ToLower(tc.ID)

	mode := tc.Mode
	if mode == "" {
		mode = "sales"
	}

	var replies []string
	leadData := map[string]string{}
	var storeHist []store.Message

	for _, userText := range tc.Turns {
		system, kb, err := r.buildBundledPrompt(convID, mode, leadData)
		if err != nil {
			return CaseResult{}, err
		}
		msgs := receptionist.BuildBundledSupportMessages(system, kb, storeHist, userText)

		callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		raw, err := r.AI.Complete(callCtx, msgs, r.Cfg.LeadTrackingEnabled())
		cancel()
		if err != nil {
			return CaseResult{}, err
		}

		reply := strings.TrimSpace(raw)
		if r.Cfg.LeadTrackingEnabled() {
			if parsed, err := ai.DecodeStructured(raw); err == nil && parsed != nil {
				reply = strings.TrimSpace(parsed.Reply)
				leadData = lead.Merge(leadData, parsed.LeadUpdates)
			}
		}
		reply = receptionist.FinalizeCustomerReply(
			reply, userText, r.Cfg.BusinessName, r.Cfg.DisplayOwnerName(), r.Cfg.BusinessDescription, nil,
		)
		replies = append(replies, reply)
		storeHist = append(storeHist, store.Message{Role: "user", Message: userText})
		storeHist = append(storeHist, store.Message{Role: "assistant", Message: reply})
	}

	verdict, note := tc.Check(replies)
	return CaseResult{Case: tc, Replies: replies, Verdict: verdict, Note: note}, nil
}

func (r *Runner) buildBundledPrompt(convID, mode string, leadData map[string]string) (system, knowledge string, err error) {
	stack, err := r.PromptBuilder.BuildSupportStack(convID, mode)
	if err != nil {
		return "", "", err
	}
	kb, err := r.PromptBuilder.KnowledgeBase()
	if err != nil {
		return "", "", err
	}
	p := r.PromptTpl
	p = strings.ReplaceAll(p, "{{business_name}}", r.Cfg.BusinessName)
	p = strings.ReplaceAll(p, "{{business_description}}", r.Cfg.BusinessDescription)
	p = strings.ReplaceAll(p, "{{your_name}}", r.Cfg.DisplayOwnerName())

	var b strings.Builder
	b.WriteString(stack)
	b.WriteString("\n\n---\n\n")
	b.WriteString(p)
	if r.StyleExtra != "" {
		b.WriteString("\n\n## Style examples\n")
		b.WriteString(r.StyleExtra)
		b.WriteString("\n")
	}
	if r.Cfg.LeadTrackingEnabled() {
		missing := lead.Missing(leadData)
		leadJSON, _ := json.Marshal(leadData)
		missJSON, _ := json.Marshal(missing)
		b.WriteString("\n\n## Runtime context\nmissing_fields: ")
		b.Write(missJSON)
		b.WriteString("\ncurrent_lead_data: ")
		b.Write(leadJSON)
		b.WriteString("\n")
	}
	return b.String(), kb, nil
}
