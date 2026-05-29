package whatsapp

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func SendText(ctx context.Context, wa *Client, chat types.JID, text string) error {
	if testSendText != nil {
		return testSendText(ctx, chat, text)
	}
	resp, err := wa.WM.SendMessage(ctx, chat, &waE2E.Message{
		Conversation: proto.String(text),
	})
	if err != nil {
		return err
	}
	if wa.Sent != nil {
		wa.Sent.Mark(resp.ID)
	}
	return nil
}

const waChunkSize = 3800

// SendTextChunked splits long messages for WhatsApp limits.
func SendTextChunked(ctx context.Context, wa *Client, chat types.JID, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	for len(text) > 0 {
		chunk := text
		if len(chunk) > waChunkSize {
			chunk = text[:waChunkSize]
			if idx := strings.LastIndex(chunk, "\n"); idx > waChunkSize/2 {
				chunk = text[:idx+1]
			}
		}
		if err := SendText(ctx, wa, chat, chunk); err != nil {
			return err
		}
		text = strings.TrimPrefix(text, chunk)
	}
	return nil
}

func PhoneToJID(phone string) types.JID {
	return types.NewJID(phone, types.DefaultUserServer)
}
