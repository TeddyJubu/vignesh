package receptionist

import (
	"fmt"
	"strings"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"
)

// PromptBuilder composes baseline + soul + mode runbook + client instructions + facts.
type PromptBuilder struct {
	cfg            *config.Config
	store          *store.DB
	instructionsMD string
}

func NewPromptBuilder(cfg *config.Config, db *store.DB, instructionsMD string) *PromptBuilder {
	return &PromptBuilder{cfg: cfg, store: db, instructionsMD: instructionsMD}
}

func (p *PromptBuilder) Build(convID string, mode string) (string, error) {
	var b strings.Builder
	b.WriteString(baseAgentInstructions(p.cfg))

	if soul, err := p.store.GetAgentNote("identity_soul"); err != nil {
		return "", err
	} else if strings.TrimSpace(soul) != "" {
		b.WriteString("\n\n## Soul\n")
		b.WriteString(strings.TrimSpace(soul))
		b.WriteString("\n")
	}

	if rb, err := p.store.GetAgentNote(runbookKeyForMode(mode)); err != nil {
		return "", err
	} else if strings.TrimSpace(rb) != "" {
		b.WriteString("\n\n## Mode runbook (")
		b.WriteString(mode)
		b.WriteString(")\n")
		b.WriteString(strings.TrimSpace(rb))
		b.WriteString("\n")
	}

	if md := strings.TrimSpace(p.instructionsMD); md != "" {
		b.WriteString("\n\n## Client instructions\n\n")
		b.WriteString(md)
		b.WriteString("\n")
	}

	facts, err := p.store.ListContactFacts(convID)
	if err != nil {
		return "", err
	}
	if len(facts) > 0 {
		b.WriteString("\nKnown facts about this contact:\n")
		for _, f := range facts {
			fmt.Fprintf(&b, "- %s: %s\n", f.Key, f.Value)
		}
	}
	return b.String(), nil
}

func (p *PromptBuilder) BuildWithInbound(convID string, in whatsapp.InboundContext, contact *store.Contact, text string) (string, string, error) {
	mode := ResolveMode(contact, in, text)
	stack, err := p.Build(convID, mode)
	return stack, mode, err
}
