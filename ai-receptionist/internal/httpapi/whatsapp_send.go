package httpapi

import (
	"context"

	"ai-receptionist/internal/whatsapp"
)

func sendWhatsAppText(ctx context.Context, wa *whatsapp.Client, phone string, text string) error {
	if wa == nil {
		return nil
	}
	jid := whatsapp.PhoneToJID(phone)
	return whatsapp.SendText(ctx, wa, jid, text)
}

