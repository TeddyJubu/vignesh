package calendar

import (
	"context"
	"fmt"
	"time"

	"ai-receptionist/internal/settings"
	"ai-receptionist/internal/tools/composio"
)

// NewFromSettings prefers Composio Google Calendar when configured, then direct Google credentials, then stub.
func NewFromSettings(resolver *settings.Resolver) Calendar {
	if resolver == nil {
		return New()
	}
	cfg, err := composio.LoadConfig(resolver)
	if err != nil || !cfg.Configured() {
		return New()
	}
	client := composio.New(cfg.APIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg = client.ResolveConfig(ctx, cfg)
	if !cfg.CalendarReady() {
		fmt.Println("Calendar: stub (Composio Google Calendar not active — reconnect in Composio dashboard)")
		return New()
	}
	svc, err := composio.NewCalendarService(client, cfg)
	if err != nil {
		return New()
	}
	fmt.Println("Calendar: using Composio Google Calendar")
	return newComposioCalendar(svc)
}

// ResolveComposioConfig loads and resolves Composio settings (shared by handler for email).
func ResolveComposioConfig(resolver *settings.Resolver) (composio.Config, *composio.Client) {
	if resolver == nil {
		return composio.Config{}, nil
	}
	cfg, err := composio.LoadConfig(resolver)
	if err != nil || !cfg.Configured() {
		return composio.Config{}, nil
	}
	client := composio.New(cfg.APIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg = client.ResolveConfig(ctx, cfg)
	return cfg, client
}
