package receptionist

import (
	"fmt"
	"log"
	"os"
	"time"

	"ai-receptionist/internal/store"
)

func logTurnPhase(convID, provider, phase string, start time.Time, err error) {
	latency := time.Since(start).Milliseconds()
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	fmt.Fprintf(os.Stderr, "turn conv=%s provider=%s phase=%s latency_ms=%d err=%q\n",
		convID, provider, phase, latency, errMsg)
	if err != nil {
		log.Printf("turn %s %s: %v", convID, phase, err)
	}
}

func traceTurn(db *store.DB, convID, phase string, start time.Time, err error) {
	if db == nil {
		return
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	_ = db.InsertTurnTrace(convID, phase, time.Since(start).Milliseconds(), errMsg)
}
