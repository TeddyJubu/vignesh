package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
	WM        *whatsmeow.Client
	onMessage func(*events.Message)
	Sent      *OutboundTracker

	ctx     context.Context
	pairMu  sync.Mutex
	pairing pairingState
}

func New(ctx context.Context, dbPath string, onMessage func(*events.Message)) (*Client, error) {
	dbLog := waLog.Stdout("Database", "INFO", true)
	container, err := sqlstore.New(ctx, "sqlite3", "file:"+dbPath+"?_foreign_keys=on", dbLog)
	if err != nil {
		return nil, fmt.Errorf("sqlstore: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("device: %w", err)
	}
	clientLog := waLog.Stdout("Client", "INFO", true)
	wm := whatsmeow.NewClient(device, clientLog)
	c := &Client{WM: wm, onMessage: onMessage, Sent: NewOutboundTracker(), ctx: ctx}
	wm.AddEventHandler(c.eventHandler)
	return c, nil
}

func (c *Client) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if c.onMessage != nil {
			c.onMessage(v)
		}
	case *events.Connected:
		fmt.Println("WhatsApp connected")
		c.clearPairingQR()
		c.notePairingEvent("connected", "WhatsApp connected.")
	case *events.Disconnected:
		fmt.Println("WhatsApp disconnected")
		c.notePairingEvent("disconnected", "WhatsApp disconnected — reconnecting…")
	case *events.LoggedOut:
		reason := ""
		if v != nil {
			reason = fmt.Sprintf(" (%s)", v.Reason)
		}
		fmt.Fprintf(os.Stderr, "Logged out from WhatsApp%s — starting new QR pairing...\n", reason)
		c.startPairing()
	}
}

func (c *Client) Start(ctx context.Context) error {
	c.ctx = ctx
	if c.WM.Store.ID == nil {
		c.startPairing()
		return nil
	}
	if err := c.connectLinked(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "connect:", err)
		c.startPairing()
		return nil
	}
	if !c.WM.IsLoggedIn() {
		c.startPairing()
	}
	return nil
}

func (c *Client) startPairing() {
	c.pairMu.Lock()
	defer c.pairMu.Unlock()
	go c.runPairingLoginLoop(c.ctx)
}

func (c *Client) runPairingLoginLoop(ctx context.Context) {
	for {
		if c.WM.IsLoggedIn() && c.WM.IsConnected() {
			return
		}
		c.clearStaleSession(ctx)

		qrChan, _ := c.WM.GetQRChannel(ctx)
		if err := c.WM.Connect(); err != nil {
			fmt.Fprintln(os.Stderr, "connect:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		retry := false
		for evt := range qrChan {
			if evt.Event == "code" {
				c.noteQRCode(evt.Code)
				fmt.Println("Scan this QR with WhatsApp (Linked Devices):")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("(QR also in dashboard → WhatsApp pairing, or journalctl -u ai-receptionist -n 80)")
				continue
			}
			fmt.Println("Login event:", evt.Event)
			switch evt.Event {
			case "timeout":
				c.notePairingEvent("timeout", "")
				retry = true
			case "success":
				c.clearPairingQR()
				c.notePairingEvent("success", "Linked successfully.")
				_ = c.connectLinked(ctx)
				if id := c.WM.Store.ID; id != nil {
					fmt.Println("Session linked — listening for messages")
					fmt.Println("Linked account JID:", id.String())
				}
				return
			}
		}
		if c.WM.IsLoggedIn() {
			_ = c.connectLinked(ctx)
			return
		}
		c.WM.Disconnect()
		if !retry {
			time.Sleep(5 * time.Second)
			retry = true
		}
		time.Sleep(2 * time.Second)
	}
}

func (c *Client) clearStaleSession(ctx context.Context) {
	if c.WM.Store.ID == nil {
		return
	}
	if c.WM.IsLoggedIn() {
		return
	}
	// Important: do NOT call Logout() here.
	// If the session is temporarily disconnected / in a bad state, calling Logout can revoke the
	// linked-device session and force a QR re-pair. Instead, disconnect and let the pairing loop
	// obtain a QR only when WhatsApp requires it.
	fmt.Fprintln(os.Stderr, "WhatsApp session not logged in — attempting re-pair without revoking session...")
	c.WM.Disconnect()
	time.Sleep(500 * time.Millisecond)
}

func (c *Client) connectLinked(ctx context.Context) error {
	if c.WM.IsConnected() && c.WM.IsLoggedIn() {
		return nil
	}
	if err := c.WM.Connect(); err != nil && !errors.Is(err, whatsmeow.ErrAlreadyConnected) {
		return err
	}
	return nil
}

func (c *Client) Disconnect() {
	if c.WM != nil {
		c.WM.Disconnect()
	}
}
