#!/usr/bin/env python3
"""Add localhost-only group management HTTP API to whatsmeow-bridge."""
from __future__ import annotations

from pathlib import Path

MAIN = Path("/opt/whatsmeow-bridge/main.go")
MARKER = "handleGroupsCreate"


def main() -> None:
    text = MAIN.read_text(encoding="utf-8")
    if MARKER in text:
        print("already patched")
        return

    routes = """
\tmux.HandleFunc("/groups/create", handleGroupsCreate)
\tmux.HandleFunc("/groups/participants/add", handleGroupsParticipantsAdd)
\tmux.HandleFunc("/groups/topic", handleGroupsSetTopic)
"""
    text = text.replace(
        'mux.HandleFunc("/inject", handleInject)',
        'mux.HandleFunc("/inject", handleInject)' + routes,
        1,
    )

    handlers = r'''
func requireLocalhost(w http.ResponseWriter, r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if host != "127.0.0.1" && host != "::1" && host != "localhost" {
		jsonError(w, "groups API only allowed from localhost", 403)
		return false
	}
	return true
}

func phonesToJIDs(phones []string) ([]types.JID, error) {
	var out []types.JID
	for _, p := range phones {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		j, err := parseJID(p)
		if err != nil {
			return nil, fmt.Errorf("invalid participant %q: %w", p, err)
		}
		out = append(out, j)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one participant required")
	}
	return out, nil
}

func handleGroupsCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if !requireLocalhost(w, r) {
		return
	}
	if connState != "connected" {
		jsonError(w, "WhatsApp not connected", 503)
		return
	}
	var req struct {
		Name         string   `json:"name"`
		Participants []string `json:"participants"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid JSON", 400)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		jsonError(w, "name required", 400)
		return
	}
	if len(name) > 25 {
		name = name[:25]
	}
	participants, err := phonesToJIDs(req.Participants)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	info, err := client.CreateGroup(ctx, whatsmeow.ReqCreateGroup{
		Name:         name,
		Participants: participants,
	})
	if err != nil {
		jsonError(w, "CreateGroup failed: "+err.Error(), 500)
		return
	}
	log.Printf("👥 Created group %s (%s) participants=%d", name, info.JID.String(), len(participants))
	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"groupJid": info.JID.String(),
		"name":     info.Name,
	})
}

func handleGroupsParticipantsAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if !requireLocalhost(w, r) {
		return
	}
	if connState != "connected" {
		jsonError(w, "WhatsApp not connected", 503)
		return
	}
	var req struct {
		GroupJID     string   `json:"groupJid"`
		Participants []string `json:"participants"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid JSON", 400)
		return
	}
	gjid, err := parseJID(req.GroupJID)
	if err != nil {
		jsonError(w, "Invalid groupJid: "+err.Error(), 400)
		return
	}
	participants, err := phonesToJIDs(req.Participants)
	if err != nil {
		jsonError(w, err.Error(), 400)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	_, err = client.UpdateGroupParticipants(ctx, gjid, participants, whatsmeow.ParticipantChangeAdd)
	if err != nil {
		jsonError(w, "Add participants failed: "+err.Error(), 500)
		return
	}
	log.Printf("👥 Added %d participant(s) to %s", len(participants), req.GroupJID)
	jsonResponse(w, map[string]interface{}{"success": true, "groupJid": req.GroupJID})
}

func handleGroupsSetTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if !requireLocalhost(w, r) {
		return
	}
	if connState != "connected" {
		jsonError(w, "WhatsApp not connected", 503)
		return
	}
	var req struct {
		GroupJID string `json:"groupJid"`
		Topic    string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid JSON", 400)
		return
	}
	gjid, err := parseJID(req.GroupJID)
	if err != nil {
		jsonError(w, "Invalid groupJid: "+err.Error(), 400)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = client.SetGroupTopic(ctx, gjid, "", "", strings.TrimSpace(req.Topic))
	if err != nil {
		jsonError(w, "SetGroupTopic failed: "+err.Error(), 500)
		return
	}
	jsonResponse(w, map[string]interface{}{"success": true, "groupJid": req.GroupJID})
}
'''

    anchor = "// ── Helpers ──────────────────────────────────────────────────────"
    if anchor not in text:
        raise SystemExit("helpers anchor not found")
    text = text.replace(anchor, handlers + "\n" + anchor, 1)
    MAIN.write_text(text, encoding="utf-8")
    print("ok: groups API patched into", MAIN)


if __name__ == "__main__":
    main()
