package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Calendar provides read/write appointment operations.
type Calendar interface {
	CheckAvailability(ctx context.Context, input string) (string, error)
	BookAppointment(ctx context.Context, convID, input string) (string, error)
}

// New returns Google Calendar when credentials are configured, otherwise a stub.
func New() Calendar {
	if creds := strings.TrimSpace(os.Getenv("GOOGLE_CALENDAR_CREDENTIALS")); creds != "" {
		return &googleCalendar{credentialsPath: creds, tz: loadTZ()}
	}
	return &stubCalendar{tz: loadTZ()}
}

func loadTZ() *time.Location {
	tzName := strings.TrimSpace(os.Getenv("CALENDAR_TZ"))
	if tzName == "" {
		tzName = "Asia/Singapore"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return time.FixedZone("SGT", 8*3600)
	}
	return loc
}

type stubCalendar struct {
	tz *time.Location
}

func (s *stubCalendar) CheckAvailability(ctx context.Context, input string) (string, error) {
	_ = ctx
	now := time.Now().In(s.tz)
	slots := []string{
		now.Add(24 * time.Hour).Format("Mon 3pm"),
		now.Add(26 * time.Hour).Format("Mon 5pm"),
		now.Add(72 * time.Hour).Format("Fri 11am"),
	}
	b, _ := json.Marshal(map[string]any{
		"available": true,
		"timezone":  s.tz.String(),
		"slots":     slots,
		"source":    "stub",
		"query":     strings.TrimSpace(input),
	})
	return string(b), nil
}

func (s *stubCalendar) BookAppointment(ctx context.Context, convID, input string) (string, error) {
	_ = ctx
	b, _ := json.Marshal(map[string]any{
		"booked":          false,
		"conv_id":         convID,
		"idempotency_key": input,
		"reason":          "stub_calendar_no_write",
		"timezone":        s.tz.String(),
	})
	return string(b), nil
}

// googleCalendar is a placeholder for real Google Calendar API integration.
// When GOOGLE_CALENDAR_CREDENTIALS is set, extend this type with google.golang.org/api/calendar/v3.
type googleCalendar struct {
	credentialsPath string
	tz              *time.Location
}

func (g *googleCalendar) CheckAvailability(ctx context.Context, input string) (string, error) {
	_ = ctx
	// Credentials present but full OAuth client wiring is deferred; fall back to stub slots with source tag.
	s := &stubCalendar{tz: g.tz}
	out, err := s.CheckAvailability(ctx, input)
	if err != nil {
		return "", err
	}
	var m map[string]any
	if json.Unmarshal([]byte(out), &m) == nil {
		m["source"] = "google_calendar_deferred"
		m["note"] = fmt.Sprintf("credentials at %s — implement API client", g.credentialsPath)
		b, _ := json.Marshal(m)
		return string(b), nil
	}
	return out, nil
}

func (g *googleCalendar) BookAppointment(ctx context.Context, convID, input string) (string, error) {
	b, _ := json.Marshal(map[string]any{
		"booked":          false,
		"conv_id":         convID,
		"idempotency_key": input,
		"reason":          "google_calendar_write_deferred",
	})
	return string(b), nil
}
