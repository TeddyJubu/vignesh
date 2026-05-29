package ops

import (
	"ai-receptionist/internal/aiface"
	"ai-receptionist/internal/adapters/calendar"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/tools/composio"
	"ai-receptionist/internal/whatsapp"
)

// WorkerEnv holds dependencies for async job handlers.
type WorkerEnv struct {
	Store    *store.DB
	Cfg      *config.Config
	WA       *whatsapp.Client
	AI       aiface.Provider
	Calendar calendar.Calendar
	Mailer   composio.Mailer
}
