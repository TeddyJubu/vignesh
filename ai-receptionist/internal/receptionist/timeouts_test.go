package receptionist

import (
	"testing"
	"time"

	"ai-receptionist/internal/agent"
)

func TestEnvDurationSeconds(t *testing.T) {
	t.Setenv("PLANNER_TIMEOUT_SEC", "9")
	if got := plannerTimeout(); got != 9*time.Second {
		t.Fatalf("plannerTimeout=%v", got)
	}
}

func TestIsStaleAgentState(t *testing.T) {
	if isStaleAgentState(agent.State{StartedAtUNIX: time.Now().Unix()}) {
		t.Fatal("fresh state should not be stale")
	}
	old := agent.State{StartedAtUNIX: time.Now().Add(-2 * defaultAgentStateMaxAge).Unix()}
	if !isStaleAgentState(old) {
		t.Fatal("old state should be stale")
	}
}
