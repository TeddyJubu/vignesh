package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/memory"
	"ai-receptionist/internal/settings"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/tools/composio"
	"ai-receptionist/internal/whatsapp"

	"github.com/google/uuid"
)

type Server struct {
	cfg        *config.Config
	store      *store.DB
	settings   *settings.Resolver
	distDir    string
	graphiti   *memory.Client
	httpServer *http.Server
	wa         *whatsapp.Client

	promptTpl      string
	styleExtra     string
	instructionsMD string

	invalidatePrompt func()

	pingMu    sync.Mutex
	pingCache pingCacheEntry

	pairingHub *pairingHub
}

type pingCacheEntry struct {
	at       time.Time
	provider string
	model    string
	ok       bool
	errMsg   string
}

const providerPingCacheTTL = 45 * time.Second

func New(cfg *config.Config, db *store.DB, distDir, graphitiURL string) *Server {
	return &Server{
		cfg:      cfg,
		store:    db,
		settings: settings.New(db),
		distDir:  distDir,
		graphiti: memory.NewClient(graphitiURL),
	}
}

func (s *Server) SetWhatsAppClient(c *whatsapp.Client) {
	s.wa = c
}

func (s *Server) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/settings", s.handleSettings)
	mux.HandleFunc("/api/instructions", s.handleInstructions)
	mux.HandleFunc("/api/dreams", s.handleDreams)
	mux.HandleFunc("/api/dreams/propose", s.handleDreamPropose)
	mux.HandleFunc("/api/dreams/", s.handleDreamByID)
	mux.HandleFunc("/api/memory/ingest", s.handleMemoryIngest)
	mux.HandleFunc("/api/memory/recall", s.handleMemoryRecall)
	mux.HandleFunc("/api/providers/status", s.handleProviderStatus)
	mux.HandleFunc("/api/providers/ping", s.handleProviderPing)
	mux.HandleFunc("/api/composio/status", s.handleComposioStatus)
	mux.HandleFunc("/api/auth/request-otp", s.handleAuthRequestOTP)
	mux.HandleFunc("/api/auth/verify-otp", s.handleAuthVerifyOTP)
	mux.HandleFunc("/api/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("/api/me", s.handleMe)
	mux.HandleFunc("/api/access/roles", s.handleAccessRoles)
	mux.HandleFunc("/api/access/allowlist", s.handleAccessAllowlist)
	mux.HandleFunc("/api/pairing", s.handlePairingState)
	mux.HandleFunc("/api/pairing/stream", s.handlePairingStream)
	mux.HandleFunc("/api/pairing/qr.png", s.handlePairingQR)
	mux.HandleFunc("/api/pairing/refresh", s.handlePairingRefresh)
	mux.HandleFunc("/api/pairing/unlink", s.handlePairingUnlink)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	if strings.TrimSpace(s.distDir) != "" {
		mux.Handle("/", s.spaHandler(s.distDir))
	}

	handler := s.withAuth(mux)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}()

	err := s.httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// StartPairingHub begins pairing poll + SSE publish loop (call after SetWhatsAppClient).
func (s *Server) StartPairingHub(ctx context.Context) {
	s.startPairingPollLoop(ctx)
}

// SetPromptInvalidator clears cached prompt fragments when dashboard instructions change.
func (s *Server) SetPromptInvalidator(fn func()) {
	s.invalidatePrompt = fn
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil && r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		actor := Actor{
			Phone:       "",
			Role:        "admin",
			Permissions: map[string]bool{},
			Source:      "open",
		}
		r = r.WithContext(ContextWithActor(r.Context(), actor))
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authorizeRequest(r *http.Request) bool {
	a, ok := ActorFromContext(r.Context())
	if !ok {
		return false
	}
	if a.Role == "admin" {
		return true
	}
	if a.Role != "manager" {
		return false
	}
	perm := requiredPermission(r)
	if perm == "" {
		return true
	}
	return a.Permissions != nil && a.Permissions[perm]
}

func requiredPermission(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	p := r.URL.Path
	switch {
	case p == "/api/me":
		return ""
	case p == "/api/settings":
		return "settings"
	case p == "/api/instructions":
		return "instructions"
	case p == "/api/dreams" || strings.HasPrefix(p, "/api/dreams/"):
		return "dreams"
	case strings.HasPrefix(p, "/api/providers/"):
		return "providers"
	case strings.HasPrefix(p, "/api/memory/"):
		return "memory"
	case strings.HasPrefix(p, "/api/access/"):
		return "access"
	case strings.HasPrefix(p, "/api/pairing"):
		return "" // admin-only enforced in handlers
	default:
		return ""
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListAppSettings()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		out := map[string]string{}
		for _, it := range list {
			out[it.Key] = it.Value
		}
		writeJSON(w, 200, map[string]any{"settings": out})
	case http.MethodPut:
		var body struct {
			Settings map[string]string `json:"settings"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		for k, v := range body.Settings {
			if err := s.store.UpsertAppSetting(k, v); err != nil {
				writeJSON(w, 500, map[string]any{"error": err.Error()})
				return
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleDreams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 100
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
				limit = n
			}
		}
		list, err := s.store.ListDreamProposals(limit)
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"dreams": list})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type instructionPatch struct {
	TargetKey   string `json:"target_key"`
	NewContent  string `json:"new_content"`
	UnifiedDiff string `json:"unified_diff,omitempty"`
}

func (s *Server) handleDreamByID(w http.ResponseWriter, r *http.Request) {
	// /api/dreams/:id/apply
	path := strings.TrimPrefix(r.URL.Path, "/api/dreams/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		writeJSON(w, 404, map[string]any{"error": "not found"})
		return
	}
	id := parts[0]
	action := parts[1]
	if action != "apply" || r.Method != http.MethodPost {
		writeJSON(w, 404, map[string]any{"error": "not found"})
		return
	}

	p, err := s.store.GetDreamProposal(id)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	if p == nil {
		writeJSON(w, 404, map[string]any{"error": "not found"})
		return
	}
	var patch instructionPatch
	_ = json.Unmarshal([]byte(p.Patch), &patch)
	if strings.TrimSpace(patch.TargetKey) == "" {
		writeJSON(w, 400, map[string]any{"error": "invalid patch"})
		return
	}
	if err := s.store.UpsertAgentNote(patch.TargetKey, patch.NewContent); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	if s.invalidatePrompt != nil {
		s.invalidatePrompt()
	}
	_ = s.store.UpdateDreamProposalStatus(id, "applied")
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleProviderPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	force := strings.TrimSpace(r.URL.Query().Get("force")) == "1"
	now := time.Now()

	s.pingMu.Lock()
	if !force && !s.pingCache.at.IsZero() && now.Sub(s.pingCache.at) < providerPingCacheTTL {
		cached := s.pingCache
		s.pingMu.Unlock()
		out := map[string]any{
			"ok":       cached.ok,
			"provider": cached.provider,
			"cached":   true,
		}
		if cached.model != "" {
			out["model"] = cached.model
		}
		if cached.errMsg != "" {
			out["message"] = cached.errMsg
			out["error"] = cached.errMsg
		}
		writeJSON(w, 200, out)
		return
	}
	s.pingMu.Unlock()

	st := resolveProviderStatus(s.settings, s.cfg.ResolvedAIProvider(), s.cfg.Model)
	p, err := ai.NewProviderFromSettings(s.cfg, s.settings)
	if err != nil {
		writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	pingErr := p.Ping(ctx)
	providerName := p.Name()
	if providerName == "" {
		providerName = st.Provider
	}

	entry := pingCacheEntry{
		at:       now,
		provider: providerName,
		model:    st.Model,
		ok:       pingErr == nil,
	}
	if pingErr != nil {
		entry.errMsg = pingErr.Error()
	}

	s.pingMu.Lock()
	s.pingCache = entry
	s.pingMu.Unlock()

	if pingErr != nil {
		writeJSON(w, 200, map[string]any{
			"ok":       false,
			"provider": providerName,
			"model":    st.Model,
			"error":    pingErr.Error(),
			"message":  pingErr.Error(),
		})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "provider": providerName, "model": st.Model})
}

func (s *Server) handleComposioStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	verify := strings.TrimSpace(r.URL.Query().Get("verify")) == "1"

	allowRaw, _ := s.store.GetAppSetting("composio.allowlist")
	enabledTools := []string{}
	for _, part := range strings.Split(allowRaw, ",") {
		if v := strings.TrimSpace(part); v != "" {
			enabledTools = append(enabledTools, v)
		}
	}

	integ, err := composio.BuildIntegrationStatus(r.Context(), s.settings, verify)
	integrated := integ.CalendarReady || integ.GmailReady
	out := map[string]any{
		"ok":              integrated,
		"configured":      integ.Configured,
		"enabled_tools":   enabledTools,
		"calendar_ready":  integ.CalendarReady,
		"gmail_ready":     integ.GmailReady,
		"needs_reauth":    integ.NeedsReauth,
		"expired_accounts": integ.ExpiredAccounts,
	}
	if integ.UserID != "" {
		out["user_id"] = integ.UserID
	}
	if integ.Timezone != "" {
		out["timezone"] = integ.Timezone
	}
	if integ.CalendarAccount != "" {
		out["calendar_account_id"] = integ.CalendarAccount
	}
	if integ.GmailAccount != "" {
		out["gmail_account_id"] = integ.GmailAccount
	}
	if len(integ.ConnectedAccounts) > 0 {
		out["connected_accounts"] = integ.ConnectedAccounts
	}

	if err != nil {
		out["ok"] = false
		out["message"] = err.Error()
		writeJSON(w, 200, out)
		return
	}

	if !integ.Configured {
		out["ok"] = false
		out["message"] = "Composio is not configured. Set composio.api_key."
		writeJSON(w, 200, out)
		return
	}
	if integ.VerifyError != "" {
		out["ok"] = false
		out["message"] = integ.VerifyError
		writeJSON(w, 200, out)
		return
	}

	switch {
	case integ.CalendarReady && integ.GmailReady:
		out["message"] = "Google Calendar and Gmail are live via Composio."
	case integ.CalendarReady:
		out["message"] = "Google Calendar is live. Connect Gmail in Composio for confirmation emails."
	case integ.GmailReady:
		out["message"] = "Gmail is live. Connect Google Calendar in Composio for live booking."
	case integ.NeedsReauth:
		out["ok"] = false
		out["message"] = "Composio auth configs exist but all Google connections are expired or revoked. Reconnect Google Calendar and Gmail in the Composio dashboard (Users → connect accounts)."
	default:
		out["ok"] = false
		out["message"] = "Composio API key is set. Connect Google Calendar and Gmail in the Composio dashboard."
	}
	if verify {
		out["verified"] = true
	}

	writeJSON(w, 200, out)
}

func (s *Server) spaHandler(distDir string) http.Handler {
	distDir = strings.TrimSpace(distDir)
	fileServer := http.FileServer(http.Dir(distDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, 404, map[string]any{"error": "not found"})
			return
		}
		rel := strings.TrimPrefix(filepath.Clean("/"+strings.TrimPrefix(r.URL.Path, "/")), "/")
		if rel == "" || rel == "." {
			rel = "index.html"
		}
		candidate := filepath.Join(distDir, rel)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
	})
}

func DefaultAddr() string {
	if v := strings.TrimSpace(os.Getenv("HTTP_ADDR")); v != "" {
		return v
	}
	return "127.0.0.1:8080"
}

func (s *Server) handleDreamPropose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body memory.DreamProposeRequest
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	if strings.TrimSpace(body.ConvID) == "" {
		writeJSON(w, 400, map[string]any{"error": "conv_id required"})
		return
	}

	var (
		id        string
		status    string
		title     string
		rationale string
		patch     map[string]any
	)
	if s.graphiti != nil && s.graphiti.Enabled() {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		resp, err := s.graphiti.ProposeDream(ctx, body)
		if err != nil {
			writeJSON(w, 502, map[string]any{"error": err.Error()})
			return
		}
		id = resp.ID
		status = resp.Status
		title = resp.Title
		rationale = resp.Rationale
		patch = resp.Patch
	} else {
		id = newDreamID()
		status = "proposed"
		title = strings.TrimSpace(body.Title)
		if title == "" {
			title = "Dream proposal"
		}
		rationale = strings.TrimSpace(body.Rationale)
		if rationale == "" {
			rationale = "Draft proposal (Graphiti sidecar not configured)."
		}
		patch = body.Patch
		if len(patch) == 0 {
			patch = map[string]any{
				"target_key":  "identity_soul",
				"new_content": "DRAFT: (fill in) Proposed update.",
			}
		}
	}

	patchJSON, err := normalizeDreamPatch(patch)
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": "invalid patch"})
		return
	}
	if strings.TrimSpace(status) == "" {
		status = "proposed"
	}
	if err := s.store.InsertDreamProposal(store.DreamProposal{
		ID:        id,
		Status:    status,
		Title:     title,
		Patch:     patchJSON,
		Rationale: rationale,
	}); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "status": status})
}

func newDreamID() string {
	return uuid.NewString()
}

func normalizeDreamPatch(patch map[string]any) (string, error) {
	if patch == nil {
		patch = map[string]any{}
	}
	if _, ok := patch["target_key"]; !ok {
		if t, ok := patch["target"].(string); ok && strings.TrimSpace(t) != "" {
			patch["target_key"] = t
			delete(patch, "target")
		}
	}
	if _, ok := patch["new_content"]; !ok {
		if c, ok := patch["content"].(string); ok {
			patch["new_content"] = c
			delete(patch, "content")
		}
	}
	b, err := json.Marshal(patch)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(b)) == "{}" || strings.TrimSpace(string(b)) == "null" {
		return "", fmt.Errorf("empty patch")
	}
	return string(b), nil
}

func (s *Server) handleMemoryIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.graphiti == nil || !s.graphiti.Enabled() {
		writeJSON(w, 502, map[string]any{"error": "GRAPHITI_URL not configured"})
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, 400, map[string]any{"error": "read body"})
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(s.graphiti.BaseURL(), "/")+"/ingest", strings.NewReader(string(b)))
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": "build request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.graphiti.HTTPClient().Do(req)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": "graphiti unreachable"})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 1<<20))
}

func (s *Server) handleMemoryRecall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.graphiti == nil || !s.graphiti.Enabled() {
		writeJSON(w, 502, map[string]any{"error": "GRAPHITI_URL not configured"})
		return
	}
	u := strings.TrimRight(s.graphiti.BaseURL(), "/") + "/recall?" + r.URL.Query().Encode()
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": "build request"})
		return
	}
	resp, err := s.graphiti.HTTPClient().Do(req)
	if err != nil {
		writeJSON(w, 502, map[string]any{"error": "graphiti unreachable"})
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 1<<20))
}
