package calendar

import (
	"context"

	"ai-receptionist/internal/tools/composio"
)

type composioCalendar struct {
	svc *composio.CalendarService
}

func (c *composioCalendar) CheckAvailability(ctx context.Context, input string) (string, error) {
	return c.svc.CheckAvailability(ctx, input)
}

func (c *composioCalendar) BookAppointment(ctx context.Context, convID, input string) (string, error) {
	return c.svc.BookAppointment(ctx, convID, input)
}

func newComposioCalendar(svc *composio.CalendarService) Calendar {
	if svc == nil {
		return nil
	}
	return &composioCalendar{svc: svc}
}
