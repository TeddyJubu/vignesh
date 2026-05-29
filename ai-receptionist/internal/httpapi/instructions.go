package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"ai-receptionist/internal/receptionist"
	"ai-receptionist/internal/session"
	"ai-receptionist/internal/store"
)

const sampleContactSettingKey = "instructions.sample_contact"

var dashboardRunbookKeys = []string{"julia-sales", "julia-cs", "julia-booking"}

type instructionsPayload struct {
	IdentitySoul         string            `json:"identity_soul"`
	Runbooks             map[string]string `json:"runbooks"`
	ClientInstructions   clientInstructions  `json:"client_instructions"`
	SampleContact        string            `json:"sample_contact"`
	Preview              string            `json:"preview"`
	PromptLayout         string            `json:"prompt_layout"`
}

type clientInstructions struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

// SetPromptMaterials supplies prompt.txt and file fallbacks for instruction preview.
func (s *Server) SetPromptMaterials(promptTpl, styleExtra, instructionsMD string) {
	s.promptTpl = promptTpl
	s.styleExtra = styleExtra
	s.instructionsMD = instructionsMD
}

func (s *Server) handleInstructions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sample := strings.TrimSpace(r.URL.Query().Get("sample_contact"))
		if sample == "" {
			sample, _ = s.store.GetAppSetting(sampleContactSettingKey)
		}
		out, err := s.buildInstructionsPayload(sample, "What does Epicware do?")
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, out)
	case http.MethodPut:
		var body instructionsPayload
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if err := s.saveInstructionsPayload(body); err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		if s.invalidatePrompt != nil {
			s.invalidatePrompt()
		}
		sample := strings.TrimSpace(body.SampleContact)
		previewMsg := "What does Epicware do?"
		out, err := s.buildInstructionsPayload(sample, previewMsg)
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, out)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) saveInstructionsPayload(body instructionsPayload) error {
	if err := s.store.UpsertAgentNote("identity_soul", body.IdentitySoul); err != nil {
		return err
	}
	if err := s.store.UpsertAgentNote("client_instructions", body.ClientInstructions.Content); err != nil {
		return err
	}
	for _, key := range dashboardRunbookKeys {
		if body.Runbooks == nil {
			continue
		}
		if v, ok := body.Runbooks[key]; ok {
			if err := s.store.UpsertAgentNote(key, v); err != nil {
				return fmt.Errorf("runbook %s: %w", key, err)
			}
		}
	}
	return s.store.UpsertAppSetting(sampleContactSettingKey, strings.TrimSpace(body.SampleContact))
}

func (s *Server) buildInstructionsPayload(sampleContact, previewUserMessage string) (instructionsPayload, error) {
	pb := receptionist.NewPromptBuilder(s.cfg, s.store, s.instructionsMD)

	soul, err := pb.Soul()
	if err != nil {
		return instructionsPayload{}, err
	}
	kb, err := pb.KnowledgeBase()
	if err != nil {
		return instructionsPayload{}, err
	}

	runbooks := make(map[string]string)
	for _, key := range dashboardRunbookKeys {
		v, err := s.store.GetAgentNote(key)
		if err != nil {
			return instructionsPayload{}, err
		}
		runbooks[key] = v
	}

	source := "db"
	if kb == "" {
		source = "unknown"
	} else if strings.TrimSpace(s.instructionsMD) != "" && kb == strings.TrimSpace(s.instructionsMD) {
		source = "file"
	}

	preview, err := s.buildInstructionsPreview(pb, sampleContact, previewUserMessage)
	if err != nil {
		return instructionsPayload{}, err
	}

	layout := "bundled"
	if receptionist.UseStackedPromptLayout() {
		layout = "stacked"
	}

	return instructionsPayload{
		IdentitySoul: soul,
		Runbooks:     runbooks,
		ClientInstructions: clientInstructions{
			Source:  source,
			Content: kb,
		},
		SampleContact: strings.TrimSpace(sampleContact),
		Preview:       preview,
		PromptLayout:  layout,
	}, nil
}

func (s *Server) buildInstructionsPreview(pb *receptionist.PromptBuilder, sampleContact, userMessage string) (string, error) {
	convID := strings.TrimSpace(sampleContact)
	if convID == "" {
		convID = "dashboard_preview"
	}
	mode := "sales"
	userMessage = strings.TrimSpace(userMessage)
	if userMessage == "" {
		userMessage = "What does Epicware do?"
	}

	var history []store.Message
	if convID != "dashboard_preview" {
		msgs, err := session.GetLastTurns(context.Background(), s.store, convID, 10)
		if err != nil {
			return "", err
		}
		history = msgs
	}

	if receptionist.UseStackedPromptLayout() {
		stack, err := pb.Build(convID, mode)
		if err != nil {
			return "", err
		}
		system := s.appendPromptTemplate(stack)
		return "=== STACKED LAYOUT (legacy) ===\n\n" + system, nil
	}

	stack, err := pb.BuildSupportStack(convID, mode)
	if err != nil {
		return "", err
	}
	system := s.appendPromptTemplate(stack)
	kb, err := pb.KnowledgeBase()
	if err != nil {
		return "", err
	}
	userTurn := receptionist.BuildSupportUserTurn(
		kb,
		session.FormatLastTurnsForPrompt(historyForBundle(history, userMessage), 5),
		userMessage,
	)

	var b strings.Builder
	b.WriteString("=== PROMPT LAYOUT: bundled (production default) ===\n\n")
	b.WriteString("--- SYSTEM (SOUL.md + runtime) ---\n")
	b.WriteString(system)
	b.WriteString("\n\n--- USER TURN (knowledge + history + message + TASK) ---\n")
	b.WriteString(userTurn)
	return b.String(), nil
}

func historyForBundle(history []store.Message, current string) []store.Message {
	if len(history) == 0 {
		return history
	}
	last := history[len(history)-1]
	if last.Role == "user" && strings.TrimSpace(last.Message) == strings.TrimSpace(current) {
		return history[:len(history)-1]
	}
	return history
}

func (s *Server) appendPromptTemplate(stack string) string {
	p := s.promptTpl
	if p == "" {
		return stack
	}
	p = strings.ReplaceAll(p, "{{business_name}}", s.cfg.BusinessName)
	p = strings.ReplaceAll(p, "{{business_description}}", s.cfg.BusinessDescription)
	p = strings.ReplaceAll(p, "{{your_name}}", s.cfg.DisplayOwnerName())

	var b strings.Builder
	b.WriteString(stack)
	b.WriteString("\n\n---\n\n")
	b.WriteString(p)
	if strings.TrimSpace(s.styleExtra) != "" {
		b.WriteString("\n\n## Style examples\n")
		b.WriteString(s.styleExtra)
		b.WriteString("\n")
	}
	return b.String()
}
