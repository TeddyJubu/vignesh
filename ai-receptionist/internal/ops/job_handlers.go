package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-receptionist/internal/lead/scraper"
	"ai-receptionist/internal/research"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"
)

var workerEnv WorkerEnv

// SetWorkerEnv configures dependencies for async job handlers (call once at startup).
func SetWorkerEnv(env WorkerEnv) {
	workerEnv = env
}

// DefaultJobHandlers returns built-in async job processors.
func DefaultJobHandlers(env WorkerEnv) map[string]JobHandler {
	if env.Store != nil {
		workerEnv = env
	}
	return map[string]JobHandler{
		"research_marketing": handleResearchMarketing,
		"scrape_leads":       handleScrapeLeads,
		"dispatch_webhook":   handleDispatchWebhook,
		"email_csv":          handleEmailCSV,
		"outbound_book":      handleOutboundBook,
	}
}

func handleResearchMarketing(ctx context.Context, job store.AsyncJob) (string, error) {
	var p struct {
		Query string `json:"query"`
	}
	_ = ParseJobPayload(job, &p)
	w := research.Worker{AI: workerEnv.AI}
	brief, err := w.Run(ctx, p.Query)
	if err != nil {
		return "", err
	}
	if workerEnv.WA != nil && workerEnv.Cfg != nil {
		ownerJID := whatsapp.PhoneToJID(workerEnv.Cfg.OwnerNumber)
		_ = whatsapp.SendTextChunked(ctx, workerEnv.WA, ownerJID, brief.Markdown)
	}
	return brief.Markdown, nil
}

func handleScrapeLeads(ctx context.Context, job store.AsyncJob) (string, error) {
	var p struct {
		Query    string `json:"query"`
		Source   string `json:"source"`
		Count    int    `json:"count"`
		Vertical string `json:"vertical"`
	}
	_ = ParseJobPayload(job, &p)
	query := strings.TrimSpace(p.Query)
	if query == "" {
		query = strings.TrimSpace(p.Source)
	}
	if p.Count <= 0 {
		p.Count = 10
	}
	pipe := scraper.Pipeline{AI: workerEnv.AI, Concurrency: 10}
	leads, path, err := pipe.Run(ctx, query, p.Count, p.Vertical)
	if err != nil {
		return "", err
	}

	if workerEnv.Store != nil {
		rows := make([]store.LeadContact, 0, len(leads))
		for _, l := range leads {
			rows = append(rows, store.LeadContact{
				JobID: job.ID, Name: l.Name, Company: l.Company, Email: l.Email,
				Phone: l.Phone, FitScore: l.FitScore, PitchAngle: l.PitchAngle,
				URL: l.URL, LinkedIn: l.LinkedIn, ICPMatch: l.ICPMatch,
			})
		}
		if err := workerEnv.Store.InsertLeadContacts(job.ID, rows); err != nil {
			return "", fmt.Errorf("persist leads: %w", err)
		}
	}

	emailNote := ""
	if workerEnv.Mailer != nil && workerEnv.Cfg != nil {
		to := strings.TrimSpace(os.Getenv("OWNER_EMAIL"))
		if to == "" {
			to = strings.TrimSpace(workerEnv.Cfg.OwnerNumber) + "@placeholder.local"
		}
		body, _ := os.ReadFile(path)
		subject := fmt.Sprintf("Lead scrape: %d rows", len(leads))
		emailBody := fmt.Sprintf("Lead scrape complete (%d rows).\n\nCSV:\n%s", len(leads), string(body))
		if err := workerEnv.Mailer.SendEmail(ctx, to, subject, emailBody); err != nil {
			emailNote = " (email failed: " + err.Error() + ")"
		} else {
			emailNote = " — CSV emailed"
		}
	}

	return fmt.Sprintf("Scrape done: %d leads → %s%s", len(leads), path, emailNote), nil
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
	body, err := os.ReadFile(p.CSVPath)
	if err != nil {
		return "", err
	}
	to := strings.TrimSpace(os.Getenv("OWNER_EMAIL"))
	if to == "" {
		return fmt.Sprintf("CSV at %s (set OWNER_EMAIL for delivery)", p.CSVPath), nil
	}
	if workerEnv.Mailer != nil {
		subject := "Julia lead CSV"
		emailBody := fmt.Sprintf("%s\n\n%s", p.Note, string(body))
		if err := workerEnv.Mailer.SendEmail(ctx, to, subject, emailBody); err != nil {
			return "", err
		}
		return fmt.Sprintf("Emailed CSV to %s", to), nil
	}
	return fmt.Sprintf("CSV ready at %s (mailer not configured)", p.CSVPath), nil
}

func handleOutboundBook(ctx context.Context, job store.AsyncJob) (string, error) {
	var p struct {
		ContactName    string `json:"contact_name"`
		WANumber       string `json:"wa_number"`
		MeetingPurpose string `json:"meeting_purpose"`
		OwnerConv      string `json:"owner_conv"`
	}
	_ = ParseJobPayload(job, &p)
	if strings.TrimSpace(p.WANumber) == "" {
		return "", fmt.Errorf("missing wa_number")
	}
	slots := defaultSlotOptions()
	slotsJSON, _ := json.Marshal(slots)

	req := store.BookingRequest{
		ID:             job.ID,
		OwnerConv:      p.OwnerConv,
		GuestPhone:     p.WANumber,
		GuestName:      p.ContactName,
		Status:         "awaiting_guest",
		GuestSlotsJSON: string(slotsJSON),
	}
	if workerEnv.Store != nil {
		if _, err := workerEnv.Store.InsertBookingRequest(req); err != nil {
			return "", err
		}
	}

	if workerEnv.WA != nil {
		guestJID := whatsapp.PhoneToJID(p.WANumber)
		name := p.ContactName
		if name == "" {
			name = "there"
		}
		msg := fmt.Sprintf("Hi %s — Vignesh asked me to schedule a call about %s.\n\nPick a slot:\n1) %s\n2) %s\n3) %s\n\nReply with 1, 2, or 3 (or suggest another time).",
			name, p.MeetingPurpose, slots[0], slots[1], slots[2])
		if err := whatsapp.SendText(ctx, workerEnv.WA, guestJID, msg); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("Messaged %s with 3 slot options (booking %s)", p.WANumber, job.ID[:8]), nil
}

func defaultSlotOptions() []string {
	base := time.Now().Add(24 * time.Hour)
	return []string{
		base.Format("Mon 3pm SGT"),
		base.Add(24 * time.Hour).Format("Tue 10am SGT"),
		base.Add(48 * time.Hour).Format("Wed 2pm SGT"),
	}
}
