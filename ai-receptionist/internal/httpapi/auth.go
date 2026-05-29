package httpapi

import (
	"net/http"
	"strings"
	"time"

	"ai-receptionist/internal/config"
)

func (s *Server) handleAuthRequestOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Phone string `json:"phone"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	phone := config.NormalizePhone(strings.TrimSpace(body.Phone))
	if phone == "" {
		writeJSON(w, 400, map[string]any{"error": "phone required"})
		return
	}
	role, err := s.store.GetAccessRole(phone)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	if role == nil || (role.Role != "admin" && role.Role != "manager") {
		// Avoid leaking which numbers are configured.
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}
	if s.wa == nil {
		writeJSON(w, 500, map[string]any{"error": "whatsapp not configured"})
		return
	}
	code, expiresAt, err := s.store.CreateDashboardOTP(phone, 10*time.Minute)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	msg := "Your Julia dashboard login code is: " + code + "\n\nIt expires in 10 minutes."
	_ = sendWhatsAppText(r.Context(), s.wa, phone, msg)
	writeJSON(w, 200, map[string]any{
		"ok":         true,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (s *Server) handleAuthVerifyOTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, 400, map[string]any{"error": err.Error()})
		return
	}
	phone := config.NormalizePhone(strings.TrimSpace(body.Phone))
	code := strings.TrimSpace(body.Code)
	if phone == "" || code == "" {
		writeJSON(w, 400, map[string]any{"error": "phone and code required"})
		return
	}
	ok, err := s.store.VerifyDashboardOTP(phone, code)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "invalid code"})
		return
	}
	token, sess, err := s.store.CreateDashboardSession(phone, 14*24*time.Hour)
	if err != nil {
		writeJSON(w, 403, map[string]any{"error": "not allowed"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"token": token,
		"me": map[string]any{
			"phone":       sess.Phone,
			"role":        sess.Role,
			"permissions": sess.Permissions,
			"expires_at":  sess.ExpiresAt.Format(time.RFC3339),
		},
	})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Admin bypass actors don't have a session to revoke.
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		raw := strings.TrimSpace(auth[len("bearer "):])
		_ = s.store.RevokeDashboardSession(raw)
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	a, ok := ActorFromContext(r.Context())
	if !ok {
		writeJSON(w, 401, map[string]any{"error": "unauthorized"})
		return
	}
	writeJSON(w, 200, map[string]any{
		"me": map[string]any{
			"phone":       a.Phone,
			"role":        a.Role,
			"permissions": a.Permissions,
		},
	})
}

