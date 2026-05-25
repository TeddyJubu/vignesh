package ops

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const defaultErrorLog = "errors.log"

var logMu sync.Mutex

// AppendErrorLog appends a timestamped line to errors.log (or ERROR_LOG_PATH).
func AppendErrorLog(scope string, err error) {
	if err == nil {
		return
	}
	path := os.Getenv("ERROR_LOG_PATH")
	if path == "" {
		path = defaultErrorLog
	}
	line := fmt.Sprintf("%s [%s] %v\n", time.Now().UTC().Format(time.RFC3339), scope, err)
	logMu.Lock()
	defer logMu.Unlock()
	f, openErr := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		fmt.Fprintln(os.Stderr, "errors.log:", openErr)
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}
