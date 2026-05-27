package tools

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

)

type researchMarketingTool struct{}

func (researchMarketingTool) Name() string { return "research_marketing" }
func (researchMarketingTool) Meta() Meta {
	return Meta{Description: "Queue marketing research (async)", SideEffect: SideEffectWrite, MaxLatency: 3 * time.Second}
}
func (t researchMarketingTool) Run(ctx context.Context, input string) (string, error) {
	rc := runContextFrom(ctx)
	if rc.Deps.Store == nil {
		return `{"queued":false,"reason":"no_store"}`, nil
	}
	job := AsyncJob{
		ConvID:      rc.ConvID,
		JobType:     "research_marketing",
		Payload:     fmt.Sprintf(`{"query":%q}`, strings.TrimSpace(input)),
		NotifyOwner: true,
	}
	if err := rc.Deps.Store.InsertAsyncJob(job); err != nil {
		return "", err
	}
	b, _ := json.Marshal(map[string]any{"queued": true, "job_id": job.ID, "message": "Research queued — I'll message Vignesh when ready."})
	return string(b), nil
}

type scrapeLeadsTool struct{}

func (scrapeLeadsTool) Name() string { return "scrape_leads" }
func (scrapeLeadsTool) Meta() Meta {
	return Meta{Description: "Queue lead scrape job (async)", SideEffect: SideEffectWrite, MaxLatency: 3 * time.Second}
}
func (t scrapeLeadsTool) Run(ctx context.Context, input string) (string, error) {
	rc := runContextFrom(ctx)
	if rc.Deps.Store == nil {
		return `{"queued":false,"reason":"no_store"}`, nil
	}
	job := AsyncJob{
		ConvID:      rc.ConvID,
		JobType:     "scrape_leads",
		Payload:     fmt.Sprintf(`{"source":%q}`, strings.TrimSpace(input)),
		NotifyOwner: true,
	}
	if err := rc.Deps.Store.InsertAsyncJob(job); err != nil {
		return "", err
	}
	b, _ := json.Marshal(map[string]any{"queued": true, "job_id": job.ID})
	return string(b), nil
}

type dispatchWebhookTool struct{}

func (dispatchWebhookTool) Name() string { return "dispatch_webhook" }
func (dispatchWebhookTool) Meta() Meta {
	return Meta{Description: "POST payload to configured webhook", SideEffect: SideEffectWrite, MaxLatency: 8 * time.Second}
}
func (t dispatchWebhookTool) Run(ctx context.Context, input string) (string, error) {
	rc := runContextFrom(ctx)
	url := ""
	if rc.Deps.Config != nil {
		url = strings.TrimSpace(rc.Deps.Config.WebhookURL())
	}
	if url == "" {
		return `{"dispatched":false,"reason":"no_webhook_url"}`, nil
	}
	// Minimal dispatch: enqueue as async job for reliability.
	if rc.Deps.Store != nil {
		job := AsyncJob{
			ConvID:      rc.ConvID,
			JobType:     "dispatch_webhook",
			Payload:     fmt.Sprintf(`{"url":%q,"body":%q}`, url, strings.TrimSpace(input)),
			NotifyOwner: false,
		}
		_ = rc.Deps.Store.InsertAsyncJob(job)
	}
	b, _ := json.Marshal(map[string]any{"dispatched": true, "url": url})
	return string(b), nil
}

type emailCSVTool struct{}

func (emailCSVTool) Name() string { return "email_csv_to_owner" }
func (emailCSVTool) Meta() Meta {
	return Meta{Description: "Write CSV and queue email to owner", SideEffect: SideEffectWrite, MaxLatency: 5 * time.Second}
}
func (t emailCSVTool) Run(ctx context.Context, input string) (string, error) {
	rc := runContextFrom(ctx)
	path := fmt.Sprintf("/tmp/julia-leads-%d.csv", time.Now().Unix())
	rows := parseCSVInput(input)
	if len(rows) == 0 {
		rows = [][]string{{"name", "phone", "email", "notes"}, {"example", "", "", input}}
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	w := csv.NewWriter(f)
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()
	_ = f.Close()

	if rc.Deps.Store != nil {
		payload, _ := json.Marshal(map[string]string{"csv_path": path, "note": strings.TrimSpace(input)})
		_ = rc.Deps.Store.InsertAsyncJob(AsyncJob{
			ConvID:      rc.ConvID,
			JobType:     "email_csv",
			Payload:     string(payload),
			NotifyOwner: true,
		})
	}
	b, _ := json.Marshal(map[string]any{"csv_path": path, "queued_email": true})
	return string(b), nil
}

func parseCSVInput(input string) [][]string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	var rows [][]string
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, ","))
	}
	return rows
}

func extendedRegistry() *Registry {
	return NewRegistry(
		calendarCheckTool{},
		collectEmailTool{},
		alignTimeTool{},
		bookAppointmentTool{},
		escalateTool{},
		researchMarketingTool{},
		scrapeLeadsTool{},
		dispatchWebhookTool{},
		emailCSVTool{},
	)
}
