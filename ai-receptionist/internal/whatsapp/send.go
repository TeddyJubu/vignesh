package whatsapp

import (
	"context"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func SendText(ctx context.Context, client *whatsmeow.Client, chat types.JID, text string) error {
	_, err := client.SendMessage(ctx, chat, &waE2E.Message{
		Conversation: proto.String(text),
	})
	return err
}

func PhoneToJID(phone string) types.JID {
	return types.NewJID(phone, types.DefaultUserServer)
}
