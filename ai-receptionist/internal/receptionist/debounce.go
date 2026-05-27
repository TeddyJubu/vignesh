package receptionist

import (
	"context"
	"strings"
	"sync"
	"time"

	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

type debounceJob struct {
	convID string
	lines  []string
	events []*events.Message
	ctxs   []whatsapp.InboundContext
	appCtx context.Context
}

// Debouncer batches rapid messages per conversation.
type Debouncer struct {
	seconds int
	onFlush func(context.Context, *events.Message, whatsapp.InboundContext, string)

	mu      sync.Mutex
	pending map[string]*debounceState
}

type debounceState struct {
	timer *time.Timer
	job   debounceJob
}

func NewDebouncer(seconds int, onFlush func(context.Context, *events.Message, whatsapp.InboundContext, string)) *Debouncer {
	if seconds <= 0 {
		seconds = 3
	}
	return &Debouncer{
		seconds: seconds,
		onFlush: onFlush,
		pending: make(map[string]*debounceState),
	}
}

// Cancel drops pending messages for a conversation (e.g. human takeover).
func (d *Debouncer) Cancel(convID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	st, ok := d.pending[convID]
	if !ok {
		return
	}
	if st.timer != nil {
		st.timer.Stop()
	}
	delete(d.pending, convID)
}

func (d *Debouncer) Enqueue(ctx context.Context, v *events.Message, in whatsapp.InboundContext) {
	d.mu.Lock()
	defer d.mu.Unlock()

	st, ok := d.pending[in.ConvID]
	if !ok {
		st = &debounceState{}
		d.pending[in.ConvID] = st
	}
	st.job.convID = in.ConvID
	st.job.lines = append(st.job.lines, in.Text)
	st.job.events = append(st.job.events, v)
	st.job.ctxs = append(st.job.ctxs, in)
	if st.job.appCtx == nil {
		st.job.appCtx = ctx
	}

	if st.timer != nil {
		st.timer.Stop()
	}
	convID := in.ConvID
	st.timer = time.AfterFunc(time.Duration(d.seconds)*time.Second, func() {
		d.flush(convID)
	})
}

func (d *Debouncer) flush(convID string) {
	d.mu.Lock()
	st, ok := d.pending[convID]
	if !ok {
		d.mu.Unlock()
		return
	}
	delete(d.pending, convID)
	job := st.job
	if st.timer != nil {
		st.timer.Stop()
	}
	d.mu.Unlock()

	if len(job.events) == 0 {
		return
	}
	lastEv := job.events[len(job.events)-1]
	lastIn := job.ctxs[len(job.ctxs)-1]
	combined := strings.Join(job.lines, "\n")
	parent := job.appCtx
	if parent == nil {
		parent = context.Background()
	}
	d.onFlush(parent, lastEv, lastIn, combined)
}
