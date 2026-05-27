package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"


	"ai-receptionist/internal/ai"
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/memory"
	"ai-receptionist/internal/settings"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/tools/composio"
)

type Server struct {
	cfg        *config.Config
	store      *store.DB
	settings   *settings.Resolver
	distDir    string
	graphiti   *memory.Client
	httpServer *http.Server
}

func New(cfg *config.Config, db *store.DB, distDir, graphitiURL string) *Server {
	return &Server{
		cfg:      cfg,
		store:    db,
		settings: settings.New(db),
		distDir:  distDir,
		graphiti: memory.NewClient(graphitiURL),
	}
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
	mux.HandleFunc("/api/providers/ping", s.handleProviderPing)
	mux.HandleFunc("/api/composio/status", s.handleComposioStatus)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true})
	})

	if strings.TrimSpace(s.distDir) != "" {
		mux.Handle("/", s.spaHandler(s.distDir))
	}

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
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

func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
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

func (s *Server) handleInstructions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		notes, err := s.store.ListAgentNotes()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"notes": notes})
	case http.MethodPut:
		var body struct {
			Notes map[string]string `json:"notes"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		for k, v := range body.Notes {
			if err := s.store.UpsertAgentNote(k, v); err != nil {
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
	_ = s.store.UpdateDreamProposalStatus(id, "applied")
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleProviderPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	p, err := ai.NewProviderFromSettings(s.cfg, s.settings)
	if err != nil {
		writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	if err := p.Ping(ctx); err != nil {
		writeJSON(w, 200, map[string]any{"ok": false, "provider": p.Name(), "error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "provider": p.Name()})
}

func (s *Server) handleComposioStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	key, _ := s.settings.Resolved("composio.api_key", "COMPOSIO_API_KEY")
	verify := strings.TrimSpace(r.URL.Query().Get("verify")) == "1"
	c := composio.New(key)
	out, _ := c.Status(r.Context(), verify)

	allowlistRaw, _ := s.store.GetAppSetting("composio.allowlist")
	if strings.TrimSpace(allowlistRaw) != "" {
		var list []string
		if json.Unmarshal([]byte(allowlistRaw), &list) == nil {
			out["allowlist"] = list
		}
	}
	writeJSON(w, 200, out)
}

func (s *Server) spaHandler(distDir string) http.Handler {
	distDir = strings.TrimSpace(distDir)
	fs := http.Dir(distDir)
	fileServer := http.FileServer(fs)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSON(w, 404, map[string]any{"error": "not found"})
			return
		}
		path := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if path == "." || path == "/" || path == "" {
			path = "index.html"
		}
		// If the asset exists, serve it. Otherwise fallback to index.html for SPA routes.
		if f, err := fs.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		r2 := new(http.Request)
		*r2 = *r
		r2.URL = newCopyURL(r.URL)
		r2.URL.Path = "/index.html"
		fileServer.ServeHTTP(w, r2)
	})
}

func newCopyURL(u *url.URL) *url.URL {
	c := *u
	return &c
}

func DefaultAddr() string {
	if v := strings.TrimSpace(os.Getenv("HTTP_ADDR")); v != "" {
		return v
	}
	return "127.0.0.1:8080"
}
