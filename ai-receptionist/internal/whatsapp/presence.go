package whatsapp

import (
	"context"
	"math/rand"
	"time"

	"go.mau.fi/whatsmeow/types"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// SendTyping shows composing presence, optionally waits 1–2s, then clears typing.
func SendTyping(ctx context.Context, wa *Client, chat types.JID, withDelay bool) {
	_ = wa.WM.SendChatPresence(ctx, chat, types.ChatPresenceComposing, "")
	if withDelay {
		ms := 1000 + rand.Intn(1001)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(ms) * time.Millisecond):
		}
	}
	_ = wa.WM.SendChatPresence(ctx, chat, types.ChatPresencePaused, "")
}
