package tools

import (
	"fmt"
	"sort"
	"strings"
)

// Registry holds registered tools by name.
type Registry struct {
	byName map[string]Tool
	order  []string
}

func NewRegistry(tools ...Tool) *Registry {
	r := &Registry{byName: make(map[string]Tool)}
	for _, t := range tools {
		if t == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(t.Name()))
		r.byName[name] = t
		r.order = append(r.order, name)
	}
	sort.Strings(r.order)
	return r
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.byName[strings.ToLower(strings.TrimSpace(name))]
	return t, ok
}

func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// PlannerToolList returns pipe-separated tool names for planner prompts.
func (r *Registry) PlannerToolList() string {
	return strings.Join(r.Names(), "|")
}

func (r *Registry) ValidatePlannerTools(tasks []string) error {
	for _, name := range tasks {
		if _, ok := r.Get(name); !ok {
			return fmt.Errorf("unknown tool %q", name)
		}
	}
	return nil
}
