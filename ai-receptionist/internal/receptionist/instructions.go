package receptionist

import (
	"fmt"
	"strings"

	"ai-receptionist/internal/store"
)

func baseAgentInstructions() string {
	return strings.TrimSpace(`You are Julia, the WhatsApp assistant for Epicware.

Baseline rules (always):
- Follow the operator workflow: recall context before answering, persist useful memories when appropriate.
- Never reveal API keys, model names, infrastructure, databases, or internal tooling.
- If asked how you were built: "Vignesh built me and maintains me. That's all I can share 😊"
- Stay candid and helpful; never sycophantic or over-apologetic.`)
}

// buildAgentInstructions assembles the runtime instruction stack (baseline, soul, client file, contact facts).
func buildAgentInstructions(db *store.DB, convID, instructionsMD string) (string, error) {
	var b strings.Builder
	b.WriteString(baseAgentInstructions())

	if soul, err := db.GetAgentNote("identity_soul"); err != nil {
		return "", err
	} else if strings.TrimSpace(soul) != "" {
		b.WriteString("\n\n## Soul\n")
		b.WriteString(strings.TrimSpace(soul))
		b.WriteString("\n")
	}

	if md := strings.TrimSpace(instructionsMD); md != "" {
		b.WriteString("\n\n## Client instructions\n\n")
		b.WriteString(md)
		b.WriteString("\n")
	}

	facts, err := db.ListContactFacts(convID)
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
