package store

import (
	"strings"

	"ai-receptionist/knowledge"

	_ "embed"
)

//go:embed instructions_operator.md
var instructionsOperatorRules string

func defaultIdentitySoul() string {
	return knowledge.SoulMD
}

func defaultClientInstructions() string {
	rules := strings.TrimSpace(instructionsOperatorRules)
	kb := strings.TrimSpace(knowledge.KnowledgeMD)
	if rules == "" {
		return kb
	}
	if kb == "" {
		return rules
	}
	return rules + "\n\n---\n\n" + kb
}
