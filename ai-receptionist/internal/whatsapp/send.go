package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func SendText(ctx context.Context, wa *Client, chat types.JID, text string) error {
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

func PhoneToJID(phone string) types.JID {
	return types.NewJID(phone, types.DefaultUserServer)
}
