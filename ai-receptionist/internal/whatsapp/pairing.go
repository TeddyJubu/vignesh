package whatsapp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PairingSnapshot is the canonical pairing state exposed to the dashboard API.
type PairingSnapshot struct {
	Supported   bool    `json:"supported"`
	Reachable   bool    `json:"reachable"`
	LoggedIn    *bool   `json:"logged_in"`
	Connected   *bool   `json:"connected"`
	QRAvailable bool    `json:"qr_available"`
	Event       *string `json:"event"`
	UpdatedAt   *string `json:"updated_at"`
	ExpiresAt   *string `json:"expires_at"`
	Detail      *string `json:"detail"`
}

type pairingState struct {
	mu          sync.RWMutex
	qrCode      string
	qrUpdatedAt time.Time
	qrExpiresAt time.Time
	lastEvent   string
	detail      string
	onChange    func()
}

func boolPtr(v bool) *bool { return &v }

func isoPtr(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (c *Client) SetPairingChangeHandler(fn func()) {
	if c == nil {
		return
	}
	c.pairing.mu.Lock()
	c.pairing.onChange = fn
	c.pairing.mu.Unlock()
}

func (c *Client) notifyPairingChanged() {
	c.pairing.mu.RLock()
	fn := c.pairing.onChange
	c.pairing.mu.RUnlock()
	if fn != nil {
		fn()
	}
}

func (c *Client) noteQRCode(code string) {
	if c == nil {
		return
	}
	now := time.Now()
	c.pairing.mu.Lock()
	c.pairing.qrCode = code
	c.pairing.qrUpdatedAt = now
	c.pairing.qrExpiresAt = now.Add(60 * time.Second)
	c.pairing.lastEvent = "code"
	c.pairing.detail = "Scan with WhatsApp → Linked devices"
	c.pairing.mu.Unlock()
	c.notifyPairingChanged()
}

func (c *Client) notePairingEvent(event, detail string) {
	if c == nil {
		return
	}
	c.pairing.mu.Lock()
	c.pairing.lastEvent = event
	if event != "code" {
		c.pairing.qrCode = ""
	}
	if detail != "" {
		c.pairing.detail = detail
	} else if event == "timeout" {
		c.pairing.detail = "QR expired — a new code will appear shortly."
	}
	c.pairing.mu.Unlock()
	c.notifyPairingChanged()
}

func (c *Client) clearPairingQR() {
	if c == nil {
		return
	}
	c.pairing.mu.Lock()
	c.pairing.qrCode = ""
	c.pairing.mu.Unlock()
	c.notifyPairingChanged()
}

// PairingSnapshot returns current pairing state for the dashboard.
func (c *Client) PairingSnapshot() PairingSnapshot {
	if c == nil || c.WM == nil {
		return PairingSnapshot{Supported: false, Reachable: false}
	}
	loggedIn := c.WM.IsLoggedIn()
	connected := c.WM.IsConnected()

	c.pairing.mu.RLock()
	qr := c.pairing.qrCode
	updated := c.pairing.qrUpdatedAt
	expires := c.pairing.qrExpiresAt
	event := c.pairing.lastEvent
	detail := c.pairing.detail
	c.pairing.mu.RUnlock()

	snap := PairingSnapshot{
		Supported: true,
		Reachable: true,
		LoggedIn:  boolPtr(loggedIn),
		Connected: boolPtr(connected),
	}
	if loggedIn && connected {
		snap.Detail = strPtr("WhatsApp is connected.")
		snap.QRAvailable = false
		return snap
	}
	if qr != "" {
		snap.QRAvailable = true
		snap.UpdatedAt = isoPtr(updated)
		snap.ExpiresAt = isoPtr(expires)
		if event != "" {
			snap.Event = strPtr(event)
		}
		if detail != "" {
			snap.Detail = strPtr(detail)
		}
		return snap
	}
	if loggedIn && !connected {
		snap.Detail = strPtr("Session exists but is offline — use Unlink WhatsApp or wait for reconnect.")
		return snap
	}
	if event != "" {
		snap.Event = strPtr(event)
	}
	if detail != "" {
		snap.Detail = strPtr(detail)
	} else if !loggedIn {
		snap.Detail = strPtr("Waiting for a pairing QR code…")
	}
	return snap
}

// CurrentQRCode returns the raw QR payload string (empty if none).
func (c *Client) CurrentQRCode() string {
	if c == nil {
		return ""
	}
	c.pairing.mu.RLock()
	defer c.pairing.mu.RUnlock()
	return c.pairing.qrCode
}

// RequestPairingRefresh disconnects and starts a fresh pairing loop.
func (c *Client) RequestPairingRefresh() {
	if c == nil || c.WM == nil {
		return
	}
	c.clearPairingQR()
	c.notePairingEvent("refresh", "Generating a new QR code…")
	c.WM.Disconnect()
	c.startPairing()
}

// UnlinkWhatsApp revokes the linked-device session and starts a fresh QR pairing flow.
func (c *Client) UnlinkWhatsApp(ctx context.Context) error {
	if c == nil || c.WM == nil {
		return fmt.Errorf("whatsapp not configured")
	}

	c.pairMu.Lock()
	c.pairGen++
	if c.pairCancel != nil {
		c.pairCancel()
		c.pairCancel = nil
	}
	c.pairMu.Unlock()

	c.clearPairingQR()
	c.notePairingEvent("unlink", "Unlinking WhatsApp…")

	if c.WM.IsLoggedIn() {
		if err := c.WM.Logout(ctx); err != nil {
			c.WM.Disconnect()
			if c.WM.Store != nil {
				_ = c.WM.Store.Delete(ctx)
			}
		}
	} else {
		c.WM.Disconnect()
		if c.WM.Store != nil && c.WM.Store.ID != nil {
			_ = c.WM.Store.Delete(ctx)
		}
	}

	if err := c.recreateWhatsAppClient(ctx); err != nil {
		return fmt.Errorf("recreate client: %w", err)
	}

	c.notePairingEvent("unlinked", "WhatsApp unlinked. Scan a new QR to connect.")
	c.startPairing()
	return nil
}
