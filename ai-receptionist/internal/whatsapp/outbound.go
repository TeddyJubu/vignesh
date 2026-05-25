package whatsapp

import (
	"sync"
	"time"

	"go.mau.fi/whatsmeow/types"
)

// OutboundTracker remembers message IDs the bot sent so we do not reply to our own echoes in self-chat.
type OutboundTracker struct {
	mu  sync.Mutex
	ids map[types.MessageID]time.Time
}

func NewOutboundTracker() *OutboundTracker {
	return &OutboundTracker{ids: make(map[types.MessageID]time.Time)}
}

func (t *OutboundTracker) Mark(id types.MessageID) {
	if id == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ids[id] = time.Now().Add(3 * time.Minute)
	t.gcLocked()
}

func (t *OutboundTracker) IsOurs(id types.MessageID) bool {
	if id == "" {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	exp, ok := t.ids[id]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(t.ids, id)
		return false
	}
	return true
}

func (t *OutboundTracker) gcLocked() {
	now := time.Now()
	for id, exp := range t.ids {
		if now.After(exp) {
			delete(t.ids, id)
		}
	}
}
