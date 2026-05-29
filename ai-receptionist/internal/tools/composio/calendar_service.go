package composio

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CalendarService implements calendar operations via Composio Google Calendar tools.
type CalendarService struct {
	client    *Client
	accountID string
	userID    string
	timezone  string
}

func NewCalendarService(client *Client, cfg Config) (*CalendarService, error) {
	if client == nil || !client.Configured() {
		return nil, fmt.Errorf("composio client not configured")
	}
	if strings.TrimSpace(cfg.CalendarAccountID) == "" {
		return nil, fmt.Errorf("composio calendar connected account not set")
	}
	return &CalendarService{
		client:    client,
		accountID: cfg.CalendarAccountID,
		userID:    cfg.UserID,
		timezone:  cfg.Timezone,
	}, nil
}

func (s *CalendarService) CheckAvailability(ctx context.Context, input string) (string, error) {
	query := strings.TrimSpace(input)
	text := fmt.Sprintf(
		"Find available 30-minute meeting slots for: %s. Timezone: %s. Look ahead 7 days. Return human-readable slot times.",
		query, s.timezone,
	)
	res, err := s.client.Execute(ctx, "GOOGLECALENDAR_FIND_FREE_SLOTS", ExecuteRequest{
		ConnectedAccountID: s.accountID,
		UserID:             s.userID,
		Text:               text,
	})
	if err != nil {
		return "", err
	}
	slots := extractSlots(res.Data)
	out := map[string]any{
		"available": len(slots) > 0 || res.Successful,
		"timezone":  s.timezone,
		"slots":     slots,
		"source":    "composio",
		"query":     query,
	}
	if !res.Successful && res.Error != "" {
		out["note"] = res.Error
	}
	if len(slots) == 0 && res.Data != nil {
		out["raw"] = res.Data
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

func (s *CalendarService) BookAppointment(ctx context.Context, convID, input string) (string, error) {
	query := strings.TrimSpace(input)
	text := fmt.Sprintf(
		"Create a 30-minute Google Calendar event titled 'WhatsApp call — Epicware'. "+
			"Details: %s. Timezone: %s. Conversation ID: %s. "+
			"If an attendee email appears in the request, add them and send an invite.",
		query, s.timezone, convID,
	)
	res, err := s.client.Execute(ctx, "GOOGLECALENDAR_CREATE_EVENT", ExecuteRequest{
		ConnectedAccountID: s.accountID,
		UserID:             s.userID,
		Text:               text,
	})
	if err != nil {
		return "", err
	}
	booked := res.Successful
	eventID := extractString(res.Data, "event_id", "id", "eventId")
	start := extractString(res.Data, "start", "start_datetime", "start_time")
	if !booked && res.Error != "" {
		b, _ := json.Marshal(map[string]any{
			"booked":          false,
			"conv_id":         convID,
			"idempotency_key": query,
			"reason":          res.Error,
			"source":          "composio",
		})
		return string(b), nil
	}
	out := map[string]any{
		"booked":          booked,
		"conv_id":         convID,
		"idempotency_key": query,
		"source":          "composio",
	}
	if eventID != "" {
		out["event_id"] = eventID
	}
	if start != "" {
		out["start"] = start
	}
	if res.Data != nil {
		out["raw"] = res.Data
	}
	b, _ := json.Marshal(out)
	return string(b), nil
}

func extractSlots(data map[string]any) []string {
	if data == nil {
		return nil
	}
	for _, key := range []string{"slots", "free_slots", "available_slots", "suggested_slots"} {
		if v, ok := data[key]; ok {
			if ss := stringifyList(v); len(ss) > 0 {
				return ss
			}
		}
	}
	// Nested under response_data
	if rd, ok := data["response_data"].(map[string]any); ok {
		return extractSlots(rd)
	}
	return nil
}

func extractString(data map[string]any, keys ...string) string {
	if data == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := data[k]; ok {
			if s := fmt.Sprint(v); strings.TrimSpace(s) != "" && s != "<nil>" {
				return strings.TrimSpace(s)
			}
		}
	}
	if rd, ok := data["response_data"].(map[string]any); ok {
		return extractString(rd, keys...)
	}
	return ""
}

func stringifyList(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
