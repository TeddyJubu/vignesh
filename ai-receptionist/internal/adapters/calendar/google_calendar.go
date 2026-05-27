package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func newGoogleCalendarService(ctx context.Context, credentialsPath, calendarID string, tz *time.Location) (*calendar.Service, string, error) {
	credentialsPath = strings.TrimSpace(credentialsPath)
	if credentialsPath == "" {
		return nil, "", fmt.Errorf("credentials path empty")
	}
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, "", err
	}
	creds, err := google.CredentialsFromJSON(ctx, b, calendar.CalendarScope)
	if err != nil {
		return nil, "", err
	}
	svc, err := calendar.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, "", err
	}
	if calendarID == "" {
		calendarID = "primary"
	}
	return svc, calendarID, nil
}

func (g *googleCalendar) checkAvailabilityReal(ctx context.Context, input string) (string, error) {
	svc, calID, err := newGoogleCalendarService(ctx, g.credentialsPath, g.calendarID, g.tz)
	if err != nil {
		return "", err
	}
	now := time.Now().In(g.tz)
	end := now.Add(7 * 24 * time.Hour)
	fb, err := svc.Freebusy.Query(&calendar.FreeBusyRequest{
		TimeMin: now.Format(time.RFC3339),
		TimeMax: end.Format(time.RFC3339),
		Items:   []*calendar.FreeBusyRequestItem{{Id: calID}},
	}).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	var busy []*calendar.TimePeriod
	if item, ok := fb.Calendars[calID]; ok {
		busy = item.Busy
	}
	slots := suggestFreeSlots(now, end, busy, g.tz, 3)
	b, _ := json.Marshal(map[string]any{
		"available": len(slots) > 0,
		"timezone":  g.tz.String(),
		"slots":     slots,
		"source":    "google_calendar",
		"query":     strings.TrimSpace(input),
	})
	return string(b), nil
}

type timeInterval struct{ start, end time.Time }

func suggestFreeSlots(start, end time.Time, busy []*calendar.TimePeriod, loc *time.Location, max int) []string {
	var blocks []timeInterval
	for _, b := range busy {
		if b == nil {
			continue
		}
		bs, _ := time.Parse(time.RFC3339, b.Start)
		be, _ := time.Parse(time.RFC3339, b.End)
		blocks = append(blocks, timeInterval{bs.In(loc), be.In(loc)})
	}
	var slots []string
	cursor := start.In(loc)
	for len(slots) < max && cursor.Before(end) {
		// business hours 10-18 local
		if cursor.Hour() < 10 {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 10, 0, 0, 0, loc)
		}
		if cursor.Hour() >= 18 {
			cursor = time.Date(cursor.Year(), cursor.Month(), cursor.Day()+1, 10, 0, 0, 0, loc)
			continue
		}
		slotEnd := cursor.Add(30 * time.Minute)
		if !overlapsBusy(cursor, slotEnd, blocks) {
			slots = append(slots, cursor.Format("Mon 3:04pm"))
			cursor = cursor.Add(90 * time.Minute)
			continue
		}
		cursor = cursor.Add(30 * time.Minute)
	}
	return slots
}

func overlapsBusy(start, end time.Time, busy []timeInterval) bool {
	for _, b := range busy {
		if start.Before(b.end) && end.After(b.start) {
			return true
		}
	}
	return false
}

func (g *googleCalendar) bookAppointmentReal(ctx context.Context, convID, input string) (string, error) {
	svc, calID, err := newGoogleCalendarService(ctx, g.credentialsPath, g.calendarID, g.tz)
	if err != nil {
		return "", err
	}
	start := time.Now().In(g.tz).Add(24 * time.Hour).Truncate(30 * time.Minute)
	if strings.Contains(strings.ToLower(input), "fri") {
		for start.Weekday() != time.Friday {
			start = start.Add(24 * time.Hour)
		}
	}
	end := start.Add(30 * time.Minute)
	ev := &calendar.Event{
		Summary:     "Call — WhatsApp booking",
		Description: fmt.Sprintf("Booked via Julia for conv %s", convID),
		Start:       &calendar.EventDateTime{DateTime: start.Format(time.RFC3339), TimeZone: g.tz.String()},
		End:         &calendar.EventDateTime{DateTime: end.Format(time.RFC3339), TimeZone: g.tz.String()},
	}
	created, err := svc.Events.Insert(calID, ev).Context(ctx).Do()
	if err != nil {
		return "", err
	}
	idemKey := strings.TrimSpace(input)
	if i := strings.Index(idemKey, "|"); i >= 0 {
		idemKey = strings.TrimSpace(idemKey[:i])
	}
	b, _ := json.Marshal(map[string]any{
		"booked":          true,
		"event_id":        created.Id,
		"start":           start.Format(time.RFC3339),
		"conv_id":         convID,
		"idempotency_key": idemKey,
		"source":          "google_calendar",
	})
	return string(b), nil
}
