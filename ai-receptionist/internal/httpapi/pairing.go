package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"strings"
	"sync"
	"time"

	"ai-receptionist/internal/whatsapp"

	qrcode "github.com/skip2/go-qrcode"
)

type pairingHub struct {
	mu       sync.RWMutex
	last     whatsapp.PairingSnapshot
	lastJSON string
	subs     map[chan []byte]struct{}
}

func newPairingHub() *pairingHub {
	return &pairingHub{subs: make(map[chan []byte]struct{})}
}

func buildPairingPayload(wa *whatsapp.Client) whatsapp.PairingSnapshot {
	if wa == nil {
		return whatsapp.PairingSnapshot{
			Supported: false,
			Reachable: false,
			Detail:    strPtr("WhatsApp client not configured"),
		}
	}
	return wa.PairingSnapshot()
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func snapshotJSON(s whatsapp.PairingSnapshot) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func snapshotsEqual(a, b whatsapp.PairingSnapshot) bool {
	return snapshotJSON(a) == snapshotJSON(b)
}

func (h *pairingHub) publish(wa *whatsapp.Client) {
	snap := buildPairingPayload(wa)
	j := snapshotJSON(snap)

	h.mu.Lock()
	if j == h.lastJSON {
		h.mu.Unlock()
		return
	}
	h.last = snap
	h.lastJSON = j
	subs := make([]chan []byte, 0, len(h.subs))
	for ch := range h.subs {
		subs = append(subs, ch)
	}
	h.mu.Unlock()

	msg := []byte("event: pairing_changed\ndata: " + j + "\n\n")
	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (h *pairingHub) subscribe() chan []byte {
	ch := make(chan []byte, 4)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *pairingHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *pairingHub) currentJSON() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.lastJSON != "" {
		return h.lastJSON
	}
	return snapshotJSON(whatsapp.PairingSnapshot{Supported: false, Reachable: false})
}

func (s *Server) startPairingPollLoop(ctx context.Context) {
	if s.pairingHub == nil {
		s.pairingHub = newPairingHub()
	}
	if s.wa != nil {
		s.wa.SetPairingChangeHandler(func() {
			s.pairingHub.publish(s.wa)
		})
	}
	ticker := time.NewTicker(4 * time.Second)
	go func() {
		defer ticker.Stop()
		s.pairingHub.publish(s.wa)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.pairingHub.publish(s.wa)
			}
		}
	}()
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	a, ok := ActorFromContext(r.Context())
	if !ok || a.Role != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "admin required"})
		return false
	}
	return true
}

func (s *Server) handlePairingState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	writeJSON(w, 200, buildPairingPayload(s.wa))
}

func (s *Server) handlePairingQR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	if s.wa == nil {
		writeJSON(w, 503, map[string]any{"error": "whatsapp not configured"})
		return
	}
	code := strings.TrimSpace(s.wa.CurrentQRCode())
	if code == "" {
		writeJSON(w, 404, map[string]any{"error": "no qr available"})
		return
	}
	q, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	q.BackgroundColor = color.White
	q.ForegroundColor = color.Black
	png, err := q.PNG(280)
	if err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

func (s *Server) handlePairingStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, 500, map[string]any{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	snap := buildPairingPayload(s.wa)
	initPayload, _ := json.Marshal(map[string]any{
		"type":    "ready_snapshot",
		"pairing": snap,
	})
	fmt.Fprintf(w, "event: ready_snapshot\ndata: %s\n\n", initPayload)
	flusher.Flush()

	if s.pairingHub == nil {
		s.pairingHub = newPairingHub()
	}
	ch := s.pairingHub.subscribe()
	defer s.pairingHub.unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handlePairingRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	if s.wa == nil {
		writeJSON(w, 503, map[string]any{"error": "whatsapp not configured"})
		return
	}
	s.wa.RequestPairingRefresh()
	snap := buildPairingPayload(s.wa)
	if s.pairingHub != nil {
		s.pairingHub.publish(s.wa)
	}
	writeJSON(w, 200, snap)
}

func (s *Server) handlePairingUnlink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.requireAdmin(w, r) {
		return
	}
	if s.wa == nil {
		writeJSON(w, 503, map[string]any{"error": "whatsapp not configured"})
		return
	}
	if err := s.wa.UnlinkWhatsApp(r.Context()); err != nil {
		writeJSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	snap := buildPairingPayload(s.wa)
	if s.pairingHub != nil {
		s.pairingHub.publish(s.wa)
	}
	writeJSON(w, 200, snap)
}

func renderQRPNGForTest(code string) ([]byte, error) {
	q, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	q.BackgroundColor = color.White
	q.ForegroundColor = color.Black
	var buf bytes.Buffer
	if err := q.Write(280, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
