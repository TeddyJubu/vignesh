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
	runbook, err := p.cachedRunbook(mode)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(global)
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

func (p *PromptBuilder) buildGlobalStack() (string, error) {
	var b strings.Builder
	b.WriteString(baseAgentInstructions(p.cfg))

	if soul, err := p.store.GetAgentNote("identity_soul"); err != nil {
		return "", err
	} else if strings.TrimSpace(soul) != "" {
		b.WriteString("\n\n## Soul\n")
		b.WriteString(strings.TrimSpace(soul))
		b.WriteString("\n")
	}

	if note, err := p.store.GetAgentNote("client_instructions"); err != nil {
		return "", err
	} else if md := strings.TrimSpace(note); md != "" {
		b.WriteString("\n\n## Client instructions\n\n")
		b.WriteString(md)
		b.WriteString("\n")
	} else if md := strings.TrimSpace(p.instructionsMD); md != "" {
		b.WriteString("\n\n## Client instructions\n\n")
		b.WriteString(md)
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
