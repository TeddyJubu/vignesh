package scraper

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-receptionist/internal/aiface"
)

const defaultConcurrency = 10

// Lead is one enriched contact row for CSV export.
type Lead struct {
	Name       string `json:"name"`
	Company    string `json:"company"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	URL        string `json:"url"`
	LinkedIn   string `json:"linkedin"`
	FitScore   int    `json:"fit_score"`
	ICPMatch   string `json:"icp_match"`
	PitchAngle string `json:"pitch_angle"`
	Source     string `json:"source"`
}

// Pipeline runs the 5-pass lead scrape flow.
type Pipeline struct {
	AI          aiface.Provider
	Concurrency int
}

// Run executes pass1–pass5 and writes CSV to path.
func (p *Pipeline) Run(ctx context.Context, query string, count int, vertical string) (leads []Lead, csvPath string, err error) {
	if p.AI == nil {
		return nil, "", fmt.Errorf("scraper: no AI provider")
	}
	if count <= 0 {
		count = 10
	}
	if count > 100 {
		count = 100
	}
	if p.Concurrency <= 0 {
		p.Concurrency = defaultConcurrency
	}
	if vertical == "" {
		vertical = query
	}

	raw, err := p.pass1Search(ctx, query, count, vertical)
	if err != nil {
		return nil, "", fmt.Errorf("pass1: %w", err)
	}
	enriched, err := p.pass2Enrich(ctx, raw)
	if err != nil {
		return nil, "", fmt.Errorf("pass2: %w", err)
	}
	scored, err := p.pass3Score(ctx, enriched, query)
	if err != nil {
		return nil, "", fmt.Errorf("pass3: %w", err)
	}
	withPitch, err := p.pass4Pitch(ctx, scored)
	if err != nil {
		return nil, "", fmt.Errorf("pass4: %w", err)
	}
	final, err := p.pass5QA(ctx, withPitch, count)
	if err != nil {
		return nil, "", fmt.Errorf("pass5: %w", err)
	}

	path := fmt.Sprintf("/tmp/julia-scrape-%d.csv", time.Now().UnixNano())
	if err := writeCSV(path, final); err != nil {
		return nil, "", err
	}
	return final, path, nil
}

func (p *Pipeline) pass1Search(ctx context.Context, query string, count int, vertical string) ([]Lead, error) {
	prompt := fmt.Sprintf(`Generate %d realistic B2B leads for this scrape request.
Query: %q
Vertical: %q

Return JSON array only: [{"name":"","company":"","url":""}]
Use Singapore or query locale when relevant. No markdown.`, count, query, vertical)
	raw, err := p.AI.Complete(ctx, []aiface.Message{
		{Role: "system", Content: "You output valid JSON arrays only."},
		{Role: "user", Content: prompt},
	}, false)
	if err != nil {
		return nil, err
	}
	return parseLeadArray(raw)
}

func (p *Pipeline) pass2Enrich(ctx context.Context, leads []Lead) ([]Lead, error) {
	sem := make(chan struct{}, p.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	out := make([]Lead, len(leads))
	errs := make([]error, len(leads))

	for i, lead := range leads {
		wg.Add(1)
		go func(idx int, l Lead) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			prompt := fmt.Sprintf(`Enrich this lead with plausible contact fields (use empty string if unknown).
Lead: name=%q company=%q url=%q
Return one JSON object: {"name":"","company":"","email":"","phone":"","url":"","linkedin":""}`, l.Name, l.Company, l.URL)
			raw, err := p.AI.Complete(ctx, []aiface.Message{
				{Role: "system", Content: "Output one JSON object only."},
				{Role: "user", Content: prompt},
			}, false)
			if err != nil {
				mu.Lock()
				errs[idx] = err
				out[idx] = l
				mu.Unlock()
				return
			}
			enriched, err := parseLeadObject(raw)
			if err != nil {
				mu.Lock()
				errs[idx] = err
				out[idx] = l
				mu.Unlock()
				return
			}
			if enriched.Name == "" {
				enriched.Name = l.Name
			}
			if enriched.Company == "" {
				enriched.Company = l.Company
			}
			if enriched.URL == "" {
				enriched.URL = l.URL
			}
			mu.Lock()
			out[idx] = enriched
			mu.Unlock()
		}(i, lead)
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			return out, nil // partial enrich OK
		}
	}
	return out, nil
}

func (p *Pipeline) pass3Score(ctx context.Context, leads []Lead, query string) ([]Lead, error) {
	b, _ := json.Marshal(leads)
	prompt := fmt.Sprintf(`Score each lead ICP fit 1-10 for query %q.
Input: %s
Return JSON array with same length, each: {"name":"","company":"","email":"","phone":"","url":"","linkedin":"","fit_score":0,"icp_match":"one line reason"}`, query, string(b))
	raw, err := p.AI.Complete(ctx, []aiface.Message{
		{Role: "system", Content: "Output JSON array only."},
		{Role: "user", Content: prompt},
	}, false)
	if err != nil {
		return leads, nil
	}
	scored, err := parseLeadArray(raw)
	if err != nil || len(scored) == 0 {
		return leads, nil
	}
	for i := range leads {
		if i < len(scored) {
			if scored[i].FitScore > 0 {
				leads[i].FitScore = scored[i].FitScore
			}
			if scored[i].ICPMatch != "" {
				leads[i].ICPMatch = scored[i].ICPMatch
			}
			if scored[i].Email != "" {
				leads[i].Email = scored[i].Email
			}
		}
		if leads[i].FitScore == 0 {
			leads[i].FitScore = 5
		}
	}
	return leads, nil
}

func (p *Pipeline) pass4Pitch(ctx context.Context, leads []Lead) ([]Lead, error) {
	var need []int
	for i, l := range leads {
		if l.FitScore >= 7 {
			need = append(need, i)
		}
	}
	if len(need) == 0 {
		return leads, nil
	}
	sem := make(chan struct{}, p.Concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, idx := range need {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			l := leads[i]
			prompt := fmt.Sprintf(`Write a one-sentence pitch angle for outbound to %q at %q (ICP: %s).
Return JSON: {"pitch_angle":""}`, l.Name, l.Company, l.ICPMatch)
			raw, err := p.AI.Complete(ctx, []aiface.Message{
				{Role: "user", Content: prompt},
			}, false)
			if err != nil {
				return
			}
			var wrap struct {
				PitchAngle string `json:"pitch_angle"`
			}
			if json.Unmarshal([]byte(aiStrip(raw)), &wrap) == nil && wrap.PitchAngle != "" {
				mu.Lock()
				leads[i].PitchAngle = wrap.PitchAngle
				mu.Unlock()
			}
		}(idx)
	}
	wg.Wait()
	return leads, nil
}

func (p *Pipeline) pass5QA(ctx context.Context, leads []Lead, target int) ([]Lead, error) {
	b, _ := json.Marshal(leads)
	prompt := fmt.Sprintf(`QA this lead list: dedupe by company+email, drop obvious non-ICP (fit_score<4), keep best %d.
Input: %s
Return cleaned JSON array with columns name,company,email,phone,url,linkedin,fit_score,icp_match,pitch_angle`, target, string(b))
	raw, err := p.AI.Complete(ctx, []aiface.Message{
		{Role: "system", Content: "Output JSON array only. No markdown."},
		{Role: "user", Content: prompt},
	}, false)
	if err != nil {
		return dedupeLocal(leads, target), nil
	}
	cleaned, err := parseLeadArray(raw)
	if err != nil || len(cleaned) == 0 {
		return dedupeLocal(leads, target), nil
	}
	if len(cleaned) > target {
		cleaned = cleaned[:target]
	}
	return cleaned, nil
}

func dedupeLocal(leads []Lead, target int) []Lead {
	seen := map[string]struct{}{}
	var out []Lead
	for _, l := range leads {
		if l.FitScore > 0 && l.FitScore < 4 {
			continue
		}
		key := strings.ToLower(l.Company + "|" + l.Email)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, l)
		if len(out) >= target {
			break
		}
	}
	return out
}

func writeCSV(path string, leads []Lead) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"name", "company", "email", "phone", "fit_score", "pitch_angle", "url", "linkedin", "icp_match"})
	for _, l := range leads {
		_ = w.Write([]string{
			l.Name, l.Company, l.Email, l.Phone,
			strconv.Itoa(l.FitScore), l.PitchAngle, l.URL, l.LinkedIn, l.ICPMatch,
		})
	}
	w.Flush()
	return w.Error()
}

func parseLeadArray(raw string) ([]Lead, error) {
	raw = aiStrip(raw)
	var leads []Lead
	if err := json.Unmarshal([]byte(raw), &leads); err != nil {
		// try wrapping object
		var wrap struct {
			Leads []Lead `json:"leads"`
		}
		if err2 := json.Unmarshal([]byte(raw), &wrap); err2 == nil && len(wrap.Leads) > 0 {
			return wrap.Leads, nil
		}
		return nil, err
	}
	return leads, nil
}

func parseLeadObject(raw string) (Lead, error) {
	raw = aiStrip(raw)
	var l Lead
	if err := json.Unmarshal([]byte(raw), &l); err != nil {
		return Lead{}, err
	}
	return l, nil
}

func aiStrip(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
