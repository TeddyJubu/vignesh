package research

import (
	"context"
	"fmt"
	"strings"

	"ai-receptionist/internal/aiface"
)

// Worker synthesizes a structured research brief from a query.
type Worker struct {
	AI aiface.Provider
}

// Brief is the structured research output.
type Brief struct {
	ExecutiveSummary string
	KeyFindings      []string
	Sources          []string
	Markdown         string
}

// Run performs multi-step research synthesis.
func (w *Worker) Run(ctx context.Context, query string) (Brief, error) {
	if w.AI == nil {
		return Brief{}, fmt.Errorf("research: no AI provider")
	}
	q := strings.TrimSpace(query)
	if q == "" {
		q = "marketing trends"
	}

	// Step 1: gather angles
	anglesRaw, err := w.AI.Complete(ctx, []aiface.Message{
		{Role: "system", Content: "You are a market researcher. List 5 research angles as bullet points."},
		{Role: "user", Content: "Research query: " + q},
	}, false)
	if err != nil {
		return Brief{}, err
	}

	// Step 2: synthesize report
	synthRaw, err := w.AI.Complete(ctx, []aiface.Message{
		{Role: "system", Content: `Write a structured research report in markdown with sections:
## Executive Summary
## Key Findings (bullets)
## Sources (bulleted list of plausible source types/URLs)
Be specific to the query geography and industry.`},
		{Role: "user", Content: fmt.Sprintf("Query: %s\n\nResearch angles:\n%s", q, anglesRaw)},
	}, false)
	if err != nil {
		return Brief{}, err
	}

	brief := parseMarkdownBrief(synthRaw)
	if brief.ExecutiveSummary == "" {
		brief.ExecutiveSummary = strings.TrimSpace(synthRaw)
	}
	if brief.Markdown == "" {
		brief.Markdown = synthRaw
	}
	return brief, nil
}

func parseMarkdownBrief(md string) Brief {
	var b Brief
	b.Markdown = md
	lines := strings.Split(md, "\n")
	section := ""
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(strings.ToLower(trim), "## executive"):
			section = "exec"
			continue
		case strings.HasPrefix(strings.ToLower(trim), "## key"):
			section = "findings"
			continue
		case strings.HasPrefix(strings.ToLower(trim), "## source"):
			section = "sources"
			continue
		case strings.HasPrefix(trim, "##"):
			section = ""
		}
		if strings.HasPrefix(trim, "- ") || strings.HasPrefix(trim, "* ") {
			item := strings.TrimPrefix(strings.TrimPrefix(trim, "- "), "* ")
			switch section {
			case "findings":
				b.KeyFindings = append(b.KeyFindings, item)
			case "sources":
				b.Sources = append(b.Sources, item)
			}
		} else if section == "exec" && trim != "" && !strings.HasPrefix(trim, "#") {
			if b.ExecutiveSummary != "" {
				b.ExecutiveSummary += " "
			}
			b.ExecutiveSummary += trim
		}
	}
	return b
}
