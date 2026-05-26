package receptionist

import (
	"context"
	"time"
)

const (
	overallAITimeout = 30 * time.Second
	ackDelay         = 2500 * time.Millisecond
	ackCooldown      = 5 * time.Minute
)

// budgetCtx returns a child context capped by max and the parent's remaining deadline.
func budgetCtx(parent context.Context, max time.Duration) (context.Context, context.CancelFunc) {
	if dl, ok := parent.Deadline(); ok {
		if remain := time.Until(dl); remain < max {
			max = remain
		}
	}
	if max < 100*time.Millisecond {
		max = 100 * time.Millisecond
	}
	return context.WithTimeout(parent, max)
}
