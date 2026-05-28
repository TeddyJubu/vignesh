package pb

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Repo provides Day-1 PocketBase persistence. All methods no-op when the client is disabled.
type Repo struct {
	client *Client
}

func NewRepo(client *Client) *Repo {
	return &Repo{client: client}
}

func (r *Repo) enabled() bool {
	return r != nil && r.client != nil && r.client.Enabled()
}

// UpsertSession updates or creates an agent_sessions row for wa_number.
func (r *Repo) UpsertSession(ctx context.Context, waNumber, lastIntent, lastSummary string) error {
	if !r.enabled() {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	filter := fmt.Sprintf("wa_number = '%s'", escapeFilterString(waNumber))
	items, err := r.client.listRecords(ctx, "agent_sessions", filter)
	if err != nil {
		return err
	}
	fields := map[string]any{
		"wa_number":        waNumber,
		"last_intent":      lastIntent,
		"last_summary":     lastSummary,
		"last_updated_at":  now,
	}
	if len(items) > 0 {
		return r.client.patchRecord(ctx, "agent_sessions", items[0].ID, fields)
	}
	_, err = r.client.createRecord(ctx, "agent_sessions", fields)
	return err
}

// InsertJob creates an agent_jobs row with status classified.
func (r *Repo) InsertJob(ctx context.Context, waNumber, taskType string, payload map[string]any) (string, error) {
	if !r.enabled() {
		return "", nil
	}
	if payload == nil {
		payload = map[string]any{}
	}
	fields := map[string]any{
		"wa_number": waNumber,
		"task_type": taskType,
		"payload":   payload,
		"status":    "classified",
	}
	return r.client.createRecord(ctx, "agent_jobs", fields)
}

// UpdateJobStatus is a Day-1 stub (no-op when disabled).
func (r *Repo) UpdateJobStatus(ctx context.Context, recordID, status string, result map[string]any, errMsg string) error {
	if !r.enabled() || strings.TrimSpace(recordID) == "" {
		return nil
	}
	fields := map[string]any{
		"status": status,
	}
	if result != nil {
		fields["result"] = result
	}
	if errMsg != "" {
		fields["error"] = errMsg
	}
	return r.client.patchRecord(ctx, "agent_jobs", recordID, fields)
}

// UpsertLeadContact is a Day-1 stub.
func (r *Repo) UpsertLeadContact(ctx context.Context, waNumber string, fields map[string]any) error {
	if !r.enabled() {
		return nil
	}
	_ = ctx
	_ = waNumber
	_ = fields
	return nil
}

// AppendSupportLog is a Day-1 stub.
func (r *Repo) AppendSupportLog(ctx context.Context, fields map[string]any) error {
	if !r.enabled() {
		return nil
	}
	_ = ctx
	_ = fields
	return nil
}

// AppendBookingLog is a Day-1 stub.
func (r *Repo) AppendBookingLog(ctx context.Context, fields map[string]any) error {
	if !r.enabled() {
		return nil
	}
	_ = ctx
	_ = fields
	return nil
}
