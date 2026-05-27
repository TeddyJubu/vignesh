package tools

import (
	"context"
	"time"
)

// SideEffect classifies tool behavior for planner prompts.
type SideEffect int

const (
	SideEffectNone SideEffect = iota
	SideEffectRead
	SideEffectWrite
)

// Meta describes a tool for planners and audits.
type Meta struct {
	Description string
	SideEffect  SideEffect
	MaxLatency  time.Duration
}

// Tool is a named capability invoked by the agent runner.
type Tool interface {
	Name() string
	Meta() Meta
	Run(ctx context.Context, input string) (string, error)
}

// RunContext carries per-conversation dependencies for tools.
type RunContext struct {
	ConvID string
	Deps   Deps
}

// Deps are injected services for tool implementations.
type Deps struct {
	Store    Store
	Config   Config
	WhatsApp WhatsApp
	Calendar Calendar
}

type Store interface {
	InsertToolRun(convID, tool, input, output, errMsg string, latencyMS int64) error
	InsertAsyncJob(job AsyncJob) (string, error)
	RecentMessages(convID string, limit int) ([]Message, error)
	PauseContact(phone string, until time.Time) error
	GetOrCreateContact(phone string) (Contact, error)
	RecentToolRuns(convID string, limit int) ([]ToolRun, error)
}

// AsyncJob is a minimal view for tool queueing.
type AsyncJob struct {
	ID          string
	ConvID      string
	JobType     string
	Payload     string
	NotifyOwner bool
}

type Config interface {
	BusinessName() string
	DisplayOwnerName() string
	OwnerNumber() string
	PauseHours() int
	WebhookURL() string
}

type WhatsApp interface {
	SendOwnerAlert(ctx context.Context, ownerPhone, text string) error
}

type Calendar interface {
	CheckAvailability(ctx context.Context, input string) (string, error)
	BookAppointment(ctx context.Context, convID, input string) (string, error)
}

type Message struct {
	Role    string
	Message string
}

type Contact struct {
	Status string
}

type ToolRun struct {
	Tool   string
	Input  string
	Output string
	Error  string
}
