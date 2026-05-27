package calendar

import (
	"context"
	"encoding/json"
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
		calID := strings.TrimSpace(os.Getenv("GOOGLE_CALENDAR_ID"))
		if calID == "" {
			calID = "primary"
		}
		return &googleCalendar{credentialsPath: creds, calendarID: calID, tz: loadTZ()}
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
		formatHumanSlot(now.Add(24 * time.Hour)),
		formatHumanSlot(now.Add(26 * time.Hour)),
		formatHumanSlot(now.Add(72 * time.Hour)),
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

type googleCalendar struct {
	credentialsPath string
	calendarID      string
	tz              *time.Location
}

func (g *googleCalendar) CheckAvailability(ctx context.Context, input string) (string, error) {
	if out, err := g.checkAvailabilityReal(ctx, input); err == nil {
		return out, nil
	}
	s := &stubCalendar{tz: g.tz}
	out, err := s.CheckAvailability(ctx, input)
	if err != nil {
		return "", err
	}
	var m map[string]any
	if json.Unmarshal([]byte(out), &m) == nil {
		m["source"] = "google_calendar_fallback"
		m["note"] = "Google API unavailable; showing stub slots"
		b, _ := json.Marshal(m)
		return string(b), nil
	}
	return out, nil
}

func (g *googleCalendar) BookAppointment(ctx context.Context, convID, input string) (string, error) {
	if out, err := g.bookAppointmentReal(ctx, convID, input); err == nil {
		return out, nil
	}
	b, _ := json.Marshal(map[string]any{
		"booked":          false,
		"conv_id":         convID,
		"idempotency_key": input,
		"reason":          "google_calendar_write_failed",
	})
	return string(b), nil
}
