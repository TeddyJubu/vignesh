// juliaeval runs the Epicware Julia support eval suite (see test-cases-julia-eval.md).
// Usage: CONFIG_PATH=config.json APP_DB=database.db go run ./cmd/juliaeval
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/juliaeval"
	"ai-receptionist/internal/store"
)

func main() {
	cfg, err := config.Load(envOr("CONFIG_PATH", "config.json"))
	if err != nil {
		fatal(err)
	}
	db, err := store.Open(envOr("APP_DB", "database.db"))
	if err != nil {
		fatal(err)
	}
	defer db.Close()
	if err := db.RefreshEmbeddedAgentNotes(); err != nil {
		fatal(err)
	}

	promptBytes, err := os.ReadFile(envOr("PROMPT_PATH", "prompt.txt"))
	if err != nil {
		fatal(fmt.Errorf("read prompt: %w", err))
	}
	styleExtra := ""
	if b, err := os.ReadFile(envOr("STYLE_EXAMPLES_PATH", "style-examples.txt")); err == nil {
		styleExtra = strings.TrimSpace(string(b))
	}
	instructionsMD := ""
	if b, err := os.ReadFile(envOr("INSTRUCTIONS_PATH", "knowledge/instructions.md")); err == nil {
		instructionsMD = strings.TrimSpace(string(b))
	}

	runner, err := juliaeval.NewRunner(cfg, db, string(promptBytes), styleExtra, instructionsMD)
	if err != nil {
		fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	results, err := runner.RunAll(ctx)
	if err != nil {
		fatal(err)
	}

	var pass, partial, fail, critFail int
	for _, res := range results {
		switch res.Verdict {
		case juliaeval.Pass:
			pass++
		case juliaeval.Partial:
			partial++
		default:
			fail++
			if res.Case.Critical {
				critFail++
			}
		}
		last := res.Replies[len(res.Replies)-1]
		if len(last) > 220 {
			last = last[:220] + "…"
		}
		note := res.Note
		if note != "" {
			note = " — " + note
		}
		fmt.Printf("%s %-7s %s%s\n   → %s\n", res.Case.ID, res.Verdict, res.Case.Category, note, last)
	}

	total := len(results)
	fmt.Printf("\nScore: %d/%d pass, %d partial, %d fail", pass, total, partial, fail)
	if critFail > 0 {
		fmt.Printf(" (%d critical escalation fails)", critFail)
	}
	fmt.Println()

	// Target: 25/27 pass, 0 critical fails (see test-cases-julia-eval.md).
	if critFail > 0 || fail > 2 {
		os.Exit(1)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
