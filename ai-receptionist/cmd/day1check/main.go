// day1check runs live intent classification + PocketBase writes (Day 1 manual checklist helper).
// Usage: CONFIG_PATH=... APP_DB=... go run ./cmd/day1check (from ai-receptionist module root).
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/intent"
	"ai-receptionist/internal/models"
	"ai-receptionist/internal/pb"
	"ai-receptionist/internal/settings"
	"ai-receptionist/internal/session"
	"ai-receptionist/internal/store"
)

func main() {
	cfg, err := config.Load(envOr("CONFIG_PATH", "config.json"))
	if err != nil {
		panic(err)
	}
	db, err := store.Open(envOr("APP_DB", "database.db"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	resolver := settings.New(db)
	models.SetConfigModel(cfg.Model)
	aiClient, err := ai.NewProviderForModel(cfg, resolver, models.GetModel("intent_classify"))
	if err != nil {
		panic(err)
	}
	pbRepo := pb.NewRepo(pb.NewFromEnv())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	tests := []struct{ msg, want string }{
		{"What are your pricing plans?", "support"},
		{"I want to book a meeting", "sales_qualify"},
		{"What am I doing tomorrow?", "calendar_check"},
		{"Research Meta ad trends for F&B", "research_request"},
		{"Scrape 20 dental clinics in Singapore", "lead_scrape"},
	}
	convID := "day1check_test"
	for i, tc := range tests {
		_ = db.InsertMessage(convID, "user", tc.msg)
		turns, _ := session.GetLastTurns(ctx, db, convID, 10)
		last5 := session.FormatLastTurnsForPrompt(turns, 5)
		res, err := intent.Classify(ctx, aiClient, tc.msg, last5)
		if err != nil {
			fmt.Printf("FAIL %d %q err=%v\n", i+1, tc.msg, err)
			continue
		}
		mark := "OK"
		if !strings.EqualFold(res.Intent, tc.want) {
			mark = "MISMATCH"
		}
		fmt.Printf("%s want=%s got=%s conf=%.2f summary=%q\n", mark, tc.want, res.Intent, res.Confidence, res.Summary)
		_ = pbRepo.UpsertSession(ctx, convID, res.Intent, res.Summary)
		_, _ = pbRepo.InsertJob(ctx, convID, "intent_classify", map[string]any{
			"intent": res.Intent, "confidence": res.Confidence, "message": tc.msg,
		})
	}
	// Repeat first message — prompt should include prior turns.
	tc := tests[0]
	turns, _ := session.GetLastTurns(ctx, db, convID, 10)
	last5 := session.FormatLastTurnsForPrompt(turns, 5)
	res, err := intent.Classify(ctx, aiClient, tc.msg, last5)
	if err != nil {
		fmt.Printf("REPEAT err=%v\n", err)
	} else {
		fmt.Printf("REPEAT turns_chars=%d intent=%s (history should include prior user lines)\n", len(last5), res.Intent)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
