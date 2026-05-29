package receptionist

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOverallAITimeout = 30 * time.Second
	defaultAckDelay         = 2500 * time.Millisecond
	defaultAckCooldown      = 5 * time.Minute
	defaultPlannerTimeout   = 6 * time.Second
	defaultPlannerRepair    = 4 * time.Second
	defaultToolsTimeout     = 10 * time.Second
	defaultCollateTimeout   = 20 * time.Second
	defaultFastComplete     = 20 * time.Second
	defaultAgentStateMaxAge = 60 * time.Minute
)

var (
	overallAITimeout = defaultOverallAITimeout
	ackDelay         = defaultAckDelay
	ackCooldown      = defaultAckCooldown
	agentStateMaxAge = defaultAgentStateMaxAge
)

func init() {
	overallAITimeout = envDurationSeconds("OVERALL_AI_TIMEOUT_SEC", defaultOverallAITimeout)
	ackDelay = envDurationSeconds("ACK_DELAY_SEC", defaultAckDelay)
	ackCooldown = envDurationSeconds("ACK_COOLDOWN_SEC", defaultAckCooldown)
	agentStateMaxAge = envDurationSeconds("AGENT_STATE_MAX_AGE_SEC", defaultAgentStateMaxAge)
}

func plannerTimeout() time.Duration {
	return envDurationSeconds("PLANNER_TIMEOUT_SEC", defaultPlannerTimeout)
}

func plannerRepairTimeout() time.Duration {
	return envDurationSeconds("PLANNER_REPAIR_TIMEOUT_SEC", defaultPlannerRepair)
}

func toolsTimeout() time.Duration {
	return envDurationSeconds("TOOLS_TIMEOUT_SEC", defaultToolsTimeout)
}

func collateTimeout() time.Duration {
	return envDurationSeconds("COLLATE_TIMEOUT_SEC", defaultCollateTimeout)
}

func fastCompleteTimeout() time.Duration {
	return envDurationSeconds("FAST_COMPLETE_TIMEOUT_SEC", defaultFastComplete)
}

func envDurationSeconds(key string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return time.Duration(n) * time.Second
}

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
