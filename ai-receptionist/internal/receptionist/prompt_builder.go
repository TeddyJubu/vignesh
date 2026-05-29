package receptionist

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"
)

const promptCacheTTL = 45 * time.Second

// PromptBuilder composes baseline + soul + mode runbook + client instructions + facts.
type PromptBuilder struct {
	cfg            *config.Config
	store          *store.DB
	instructionsMD string

	cacheMu sync.RWMutex
	cache   promptFragmentCache
}

type promptFragmentCache struct {
	global      string
	globalAt    time.Time
	runbooks    map[string]string
	runbooksAt  map[string]time.Time
}

func NewPromptBuilder(cfg *config.Config, db *store.DB, instructionsMD string) *PromptBuilder {
	return &PromptBuilder{
		cfg:            cfg,
		store:          db,
		instructionsMD: instructionsMD,
		cache: promptFragmentCache{
			runbooks:   make(map[string]string),
			runbooksAt: make(map[string]time.Time),
		},
	}
}

// InvalidateCache clears cached prompt fragments (e.g. after instruction updates).
func (p *PromptBuilder) InvalidateCache() {
	p.cacheMu.Lock()
	defer p.cacheMu.Unlock()
	p.cache.global = ""
	p.cache.globalAt = time.Time{}
	p.cache.runbooks = make(map[string]string)
	p.cache.runbooksAt = make(map[string]time.Time)
}

func (p *PromptBuilder) Build(convID string, mode string) (string, error) {
	global, err := p.cachedGlobalStack()
	if err != nil {
		return "", err
	}
	return p.appendRunbookAndFacts(global, convID, mode)
}

// BuildSupportStack is system content for bundled layout: SOUL + baseline + runbook + contact facts (no knowledge base).
func (p *PromptBuilder) BuildSupportStack(convID, mode string) (string, error) {
	var b strings.Builder
	soul, err := p.Soul()
	if err != nil {
		return "", err
	}
	if soul != "" {
		b.WriteString(soul)
		b.WriteString("\n\n")
	}
	b.WriteString(baseAgentInstructions(p.cfg))
	return p.appendRunbookAndFacts(b.String(), convID, mode)
}

func (p *PromptBuilder) appendRunbookAndFacts(stack, convID, mode string) (string, error) {
	runbook, err := p.cachedRunbook(mode)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(stack)
	if runbook != "" {
		b.WriteString(runbook)
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

func (p *PromptBuilder) cachedGlobalStack() (string, error) {
	now := time.Now()
	p.cacheMu.RLock()
	if p.cache.global != "" && now.Sub(p.cache.globalAt) < promptCacheTTL {
		out := p.cache.global
		p.cacheMu.RUnlock()
		return out, nil
	}
	p.cacheMu.RUnlock()

	stack, err := p.buildGlobalStack()
	if err != nil {
		return "", err
	}

	p.cacheMu.Lock()
	p.cache.global = stack
	p.cache.globalAt = now
	p.cacheMu.Unlock()
	return stack, nil
}

// Soul returns identity_soul (SOUL.md) for the system prompt.
func (p *PromptBuilder) Soul() (string, error) {
	soul, err := p.store.GetAgentNote("identity_soul")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(soul), nil
}

// KnowledgeBase returns client instructions (operational rules + KNOWLEDGE.md) for the user-turn knowledge block.
func (p *PromptBuilder) KnowledgeBase() (string, error) {
	if note, err := p.store.GetAgentNote("client_instructions"); err != nil {
		return "", err
	} else if md := strings.TrimSpace(note); md != "" {
		return md, nil
	}
	return strings.TrimSpace(p.instructionsMD), nil
}

func (p *PromptBuilder) buildGlobalStack() (string, error) {
	var b strings.Builder
	b.WriteString(baseAgentInstructions(p.cfg))

	if soul, err := p.Soul(); err != nil {
		return "", err
	} else if soul != "" {
		b.WriteString("\n\n## Soul\n")
		b.WriteString(soul)
		b.WriteString("\n")
	}

	if kb, err := p.KnowledgeBase(); err != nil {
		return "", err
	} else if kb != "" {
		b.WriteString("\n\n## Client instructions\n\n")
		b.WriteString(kb)
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (p *PromptBuilder) cachedRunbook(mode string) (string, error) {
	now := time.Now()
	p.cacheMu.RLock()
	if rb, ok := p.cache.runbooks[mode]; ok && now.Sub(p.cache.runbooksAt[mode]) < promptCacheTTL {
		p.cacheMu.RUnlock()
		return rb, nil
	}
	p.cacheMu.RUnlock()

	rb, err := p.buildRunbook(mode)
	if err != nil {
		return "", err
	}

	p.cacheMu.Lock()
	p.cache.runbooks[mode] = rb
	p.cache.runbooksAt[mode] = now
	p.cacheMu.Unlock()
	return rb, nil
}

func (p *PromptBuilder) buildRunbook(mode string) (string, error) {
	rb, err := p.store.GetAgentNote(runbookKeyForMode(mode))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(rb) == "" {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("\n\n## Mode runbook (")
	b.WriteString(mode)
	b.WriteString(")\n")
	b.WriteString(strings.TrimSpace(rb))
	b.WriteString("\n")
	return b.String(), nil
}

func (p *PromptBuilder) BuildWithInbound(convID string, in whatsapp.InboundContext, contact *store.Contact, text string) (string, string, error) {
	mode := ResolveMode(contact, in, text)
	stack, err := p.Build(convID, mode)
	return stack, mode, err
}
