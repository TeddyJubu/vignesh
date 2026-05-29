package receptionist

import (
	"testing"
	"time"

	"ai-receptionist/internal/agent"
)

func TestCompleteWithPlanner_StaleStateIgnored(t *testing.T) {
	old := agent.State{
		Plan: agent.Plan{
			Questions: []string{"What day works?"},
		},
		StartedAtUNIX: time.Now().Add(-3 * defaultAgentStateMaxAge).Unix(),
	}
	if !isStaleAgentState(old) {
		t.Fatal("expected stale")
	}
}
