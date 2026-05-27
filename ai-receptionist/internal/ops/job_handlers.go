package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-receptionist/internal/store"
)

// DefaultJobHandlers returns built-in async job processors.
func DefaultJobHandlers() map[string]JobHandler {
	return map[string]JobHandler{
		"research_marketing": handleResearchMarketing,
		"scrape_leads":       handleScrapeLeads,
		"dispatch_webhook":   handleDispatchWebhook,
		"email_csv":          handleEmailCSV,
	}
}

func handleResearchMarketing(ctx context.Context, job store.AsyncJob) (string, error) {
	var p struct {
		Query string `json:"query"`
	}
	_ = ParseJobPayload(job, &p)
	q := strings.TrimSpace(p.Query)
	if q == "" {
		q = "marketing"
	}
	// Bounded placeholder: real web search can replace this via Composio or search API.
	brief := fmt.Sprintf("Research brief for %q:\n- Focus: local SEO, GBP, conversion landing pages\n- Next step: review top 3 competitors in Singapore\n- Suggested action: run ads test with clear CTA", q)
	return brief, nil
}

func handleScrapeLeads(ctx context.Context, job store.AsyncJob) (string, error) {
	var p struct {
		Source string `json:"source"`
	}
	_ = ParseJobPayload(job, &p)
	path := fmt.Sprintf("/tmp/julia-scrape-%d.csv", time.Now().Unix())
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	_, _ = f.WriteString("name,phone,email,source\n")
	_, _ = f.WriteString(fmt.Sprintf("Lead Example,,,%s\n", strings.TrimSpace(p.Source)))
	_ = f.Close()
	return fmt.Sprintf("Scrape placeholder CSV: %s (configure scraper source in payload)", path), nil
}

func handleDispatchWebhook(ctx context.Context, job store.AsyncJob) (string, error) {
	var p struct {
		URL  string `json:"url"`
		Body string `json:"body"`
	}
	_ = ParseJobPayload(job, &p)
	if strings.TrimSpace(p.URL) == "" {
		return "", fmt.Errorf("missing url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.URL, strings.NewReader(p.Body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return fmt.Sprintf("webhook status %d", resp.StatusCode), nil
}

func handleEmailCSV(ctx context.Context, job store.AsyncJob) (string, error) {
	var p struct {
		CSVPath string `json:"csv_path"`
		Note    string `json:"note"`
	}
	_ = ParseJobPayload(job, &p)
	if strings.TrimSpace(p.CSVPath) == "" {
		return "", fmt.Errorf("missing csv_path")
	}
	// Email delivery: log path; SMTP can be wired via env SMTP_* later.
	if to := strings.TrimSpace(os.Getenv("OWNER_EMAIL")); to != "" {
		return fmt.Sprintf("CSV ready at %s (email to %s not yet wired — attach manually)", p.CSVPath, to), nil
	}
	b, _ := json.Marshal(map[string]string{"csv_path": p.CSVPath, "note": p.Note})
	return string(b), nil
}
