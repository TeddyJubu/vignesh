package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow/types"
)

// test hooks for simulated WhatsApp outbound (integration tests only).

var (
	testSendText  func(ctx context.Context, chat types.JID, text string) error
	testSetTyping func(ctx context.Context, chat types.JID, typing bool)
)

// SetTestHooks redirects SendText/SetTyping to fakes. Pass nil to clear a hook.
func SetTestHooks(send func(ctx context.Context, chat types.JID, text string) error, typing func(ctx context.Context, chat types.JID, typing bool)) {
	testSendText = send
	testSetTyping = typing
}

// ClearTestHooks restores real WhatsApp sends.
func ClearTestHooks() {
	testSendText = nil
	testSetTyping = nil
}
