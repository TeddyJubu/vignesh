package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var emailInText = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)

type calendarCheckTool struct{}

func (calendarCheckTool) Name() string { return "check_calendar_availability" }
func (calendarCheckTool) Meta() Meta {
	return Meta{Description: "Read calendar availability", SideEffect: SideEffectRead, MaxLatency: 6 * time.Second}
}
func (t calendarCheckTool) Run(ctx context.Context, input string) (string, error) {
	_ = t
	if cal := toolCalendarFrom(ctx); cal != nil {
		return cal.CheckAvailability(ctx, input)
	}
	return `{"available":true,"slots":["tomorrow 3pm","tomorrow 5pm","fri 11am"],"source":"stub"}`, nil
}

type collectEmailTool struct{}

func (collectEmailTool) Name() string { return "collect_email" }
func (collectEmailTool) Meta() Meta {
	return Meta{Description: "Parse or flag missing email", SideEffect: SideEffectNone, MaxLatency: 2 * time.Second}
}
func (collectEmailTool) Run(ctx context.Context, input string) (string, error) {
	_ = ctx
	in := strings.TrimSpace(input)
	if strings.Contains(in, "@") {
		return fmt.Sprintf(`{"email":%q}`, in), nil
	}
	return `{"email":"missing"}`, nil
}

type alignTimeTool struct{}

func (alignTimeTool) Name() string { return "align_time" }
func (alignTimeTool) Meta() Meta {
	return Meta{Description: "Suggest timezone-aligned slots", SideEffect: SideEffectNone, MaxLatency: 2 * time.Second}
}
func (alignTimeTool) Run(ctx context.Context, input string) (string, error) {
	_ = ctx
	_ = input
	return `{"timezone":"Asia/Singapore","suggested_slots":["tomorrow 3pm","fri 11am"]}`, nil
}

type bookAppointmentTool struct{}

func (bookAppointmentTool) Name() string { return "book_appointment" }
func (bookAppointmentTool) Meta() Meta {
	return Meta{Description: "Book a calendar slot with idempotency", SideEffect: SideEffectWrite, MaxLatency: 10 * time.Second}
}
func (t bookAppointmentTool) Run(ctx context.Context, input string) (string, error) {
	rc := runContextFrom(ctx)
	// Idempotency: if we've already run book_appointment for this key in this conversation,
	// return the prior output instead of booking again.
	key := idempotencyKey(rc.ConvID, input)
	if rc.Deps.Store != nil {
		if runs, err := rc.Deps.Store.RecentToolRuns(rc.ConvID, 50); err == nil {
			for _, r := range runs {
				if strings.ToLower(strings.TrimSpace(r.Tool)) != "book_appointment" || strings.TrimSpace(r.Error) != "" {
					continue
				}
				var m map[string]any
				if json.Unmarshal([]byte(r.Output), &m) == nil {
					if k, _ := m["idempotency_key"].(string); strings.TrimSpace(k) == key {
						return r.Output, nil
					}
				}
			}
		}
	}

	if cal := toolCalendarFrom(ctx); cal != nil {
		out, err := cal.BookAppointment(ctx, rc.ConvID, key+"|"+strings.TrimSpace(input))
		if err != nil {
			return "", err
		}
		// Ensure idempotency_key is present for auditing/deduplication.
		var m map[string]any
		if json.Unmarshal([]byte(out), &m) == nil {
			if _, ok := m["idempotency_key"]; !ok {
				m["idempotency_key"] = key
			}
			if booked, _ := m["booked"].(bool); booked {
				maybeSendBookingEmail(ctx, rc, input, m)
			}
			if b, err := json.Marshal(m); err == nil {
				out = string(b)
			}
		}
		return out, nil
	}
	b, _ := json.Marshal(map[string]any{
		"booked":          false,
		"idempotency_key": key,
		"reason":          "no_calendar_credentials",
	})
	return string(b), nil
}

type escalateTool struct{}

func (escalateTool) Name() string { return "escalate_to_vignesh" }
func (escalateTool) Meta() Meta {
	return Meta{Description: "Alert owner and pause bot", SideEffect: SideEffectWrite, MaxLatency: 8 * time.Second}
}
func (t escalateTool) Run(ctx context.Context, input string) (string, error) {
	rc := runContextFrom(ctx)
	cfg := rc.Deps.Config
	store := rc.Deps.Store
	if store == nil || cfg == nil {
		return "", fmt.Errorf("escalation dependencies missing")
	}
	summary := strings.TrimSpace(input)
	if summary == "" {
		summary = "Escalation requested by planner."
	}
	var b strings.Builder
	b.WriteString("🚨 Julia escalation\n")
	b.WriteString(summary)
	b.WriteString("\n\nConv: ")
	b.WriteString(rc.ConvID)
	if runs, err := store.RecentToolRuns(rc.ConvID, 5); err == nil && len(runs) > 0 {
		b.WriteString("\n\nRecent tools:\n")
		for _, r := range runs {
			fmt.Fprintf(&b, "- %s: %s\n", r.Tool, truncate(r.Output+r.Error, 120))
		}
	}
	if msgs, err := store.RecentMessages(rc.ConvID, 5); err == nil && len(msgs) > 0 {
		b.WriteString("\nRecent messages:\n")
		for _, m := range msgs {
			fmt.Fprintf(&b, "- [%s] %s\n", m.Role, truncate(m.Message, 100))
		}
	}
	if wa := rc.Deps.WhatsApp; wa != nil {
		if err := wa.SendOwnerAlert(ctx, cfg.OwnerNumber(), b.String()); err != nil {
			return "", err
		}
	}
	hours := cfg.PauseHours()
	if hours <= 0 {
		hours = 24
	}
	until := time.Now().Add(time.Duration(hours) * time.Hour)
	if _, err := store.GetOrCreateContact(rc.ConvID); err != nil {
		return "", err
	}
	if err := store.PauseContact(rc.ConvID, until); err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]any{
		"escalated":        true,
		"customer_message": "I've flagged this for Vignesh — he'll follow up with you shortly.",
		"paused_until":     until.Format(time.RFC3339),
	})
	return string(out), nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func idempotencyKey(convID, input string) string {
	return convID + ":" + strings.TrimSpace(input)
}

type ctxKey struct{}

func ContextWithRun(ctx context.Context, rc RunContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, rc)
}

func runContextFrom(ctx context.Context) RunContext {
	if v, ok := ctx.Value(ctxKey{}).(RunContext); ok {
		return v
	}
	return RunContext{}
}

type calKey struct{}

func ContextWithCalendar(ctx context.Context, cal Calendar) context.Context {
	return context.WithValue(ctx, calKey{}, cal)
}

func toolCalendarFrom(ctx context.Context) Calendar {
	if v, ok := ctx.Value(calKey{}).(Calendar); ok {
		return v
	}
	rc := runContextFrom(ctx)
	return rc.Deps.Calendar
}

func toolMailerFrom(ctx context.Context) Mailer {
	rc := runContextFrom(ctx)
	return rc.Deps.Mailer
}

func maybeSendBookingEmail(ctx context.Context, rc RunContext, input string, booked map[string]any) {
	mailer := toolMailerFrom(ctx)
	if mailer == nil {
		return
	}
	to := emailInText.FindString(input)
	if to == "" {
		return
	}
	subject := "Your call with Epicware is confirmed"
	body := fmt.Sprintf("Hi,\n\nYour call slot is confirmed: %s\n\nWe look forward to speaking with you.\n\n— Julia, Epicware", strings.TrimSpace(input))
	if err := mailer.SendEmail(ctx, to, subject, body); err == nil {
		booked["email_sent"] = true
		booked["email_to"] = to
	} else {
		booked["email_sent"] = false
		booked["email_error"] = err.Error()
	}
}

// DefaultRegistry registers all receptionist tools.
func DefaultRegistry() *Registry {
	return extendedRegistry()
}
