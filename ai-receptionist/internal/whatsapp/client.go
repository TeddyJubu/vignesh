package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	c := &Client{WM: wm, onMessage: onMessage, Sent: NewOutboundTracker()}
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
	case *events.Disconnected:
		fmt.Println("WhatsApp disconnected")
	case *events.LoggedOut:
		fmt.Fprintln(os.Stderr, "Logged out — delete whatsmeow.db and scan QR again")
	}
}

func (c *Client) Start(ctx context.Context) error {
	if c.WM.Store.ID != nil {
		return c.connectLinked(ctx)
	}
	go c.runPairingLoginLoop(ctx)
	return nil
}

func (c *Client) runPairingLoginLoop(ctx context.Context) {
	for {
		if c.WM.IsLoggedIn() {
			return
		}
		qrChan, _ := c.WM.GetQRChannel(ctx)
		if err := c.WM.Connect(); err != nil {
			fmt.Fprintln(os.Stderr, "connect:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		retry := false
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("Scan this QR with WhatsApp (Linked Devices):")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				continue
			}
			fmt.Println("Login event:", evt.Event)
			switch evt.Event {
			case "timeout":
				retry = true
			case "success":
				c.connectLinked(ctx)
				return
			}
		}
		if c.WM.IsLoggedIn() {
			c.connectLinked(ctx)
			return
		}
		c.WM.Disconnect()
		if !retry {
			return
		}
		time.Sleep(2 * time.Second)
	}
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
