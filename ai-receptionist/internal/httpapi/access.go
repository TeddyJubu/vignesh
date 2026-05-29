package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"ai-receptionist/internal/config"
)

func (s *Server) handleAccessRoles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		role := strings.TrimSpace(r.URL.Query().Get("role"))
		list, err := s.store.ListAccessRoles(role, 5000)
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"roles": list})
	case http.MethodPost:
		var body struct {
			Phone       string          `json:"phone"`
			Role        string          `json:"role"`
			Permissions map[string]bool `json:"permissions"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		phone := config.NormalizePhone(strings.TrimSpace(body.Phone))
		role := strings.ToLower(strings.TrimSpace(body.Role))
		if phone == "" || role == "" {
			writeJSON(w, 400, map[string]any{"error": "phone and role required"})
			return
		}
		if err := s.store.UpsertAccessRole(phone, role, body.Permissions); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	case http.MethodDelete:
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
		if err := s.store.DeleteAccessRole(phone); err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAccessAllowlist(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		allowAllRaw, _ := s.store.GetAppSetting("access.allow_all")
		allowListRaw, _ := s.store.GetAppSetting("access.allow_list")
		allowAll := strings.TrimSpace(allowAllRaw) != "0"
		var list []string
		_ = json.Unmarshal([]byte(allowListRaw), &list)
		if list == nil {
			list = []string{}
		}
		writeJSON(w, 200, map[string]any{
			"allow_all":  allowAll,
			"allow_list": list,
		})
	case http.MethodPut:
		var body struct {
			AllowAll  *bool    `json:"allow_all"`
			AllowList []string `json:"allow_list"`
		}
		if err := readJSON(r, &body); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		if body.AllowAll != nil {
			v := "0"
			if *body.AllowAll {
				v = "1"
			}
			if err := s.store.UpsertAppSetting("access.allow_all", v); err != nil {
				writeJSON(w, 500, map[string]any{"error": err.Error()})
				return
			}
		}
		if body.AllowList != nil {
			norm := make([]string, 0, len(body.AllowList))
			seen := map[string]bool{}
			for _, p := range body.AllowList {
				n := config.NormalizePhone(strings.TrimSpace(p))
				if n == "" || seen[n] {
					continue
				}
				seen[n] = true
				norm = append(norm, n)
			}
			b, _ := json.Marshal(norm)
			if err := s.store.UpsertAppSetting("access.allow_list", string(b)); err != nil {
				writeJSON(w, 500, map[string]any{"error": err.Error()})
				return
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

