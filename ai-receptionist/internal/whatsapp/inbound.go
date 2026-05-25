package whatsapp

import (
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

// ExtractInboundText returns user-visible text (pattern from wabot inbound.go).
func ExtractInboundText(m *waE2E.Message) string {
	if m == nil {
		return ""
	}
	if t := m.GetConversation(); t != "" {
		return t
	}
	if et := m.GetExtendedTextMessage(); et != nil {
		if t := et.GetText(); t != "" {
			return t
		}
	}
	if img := m.GetImageMessage(); img != nil {
		if c := img.GetCaption(); c != "" {
			return c
		}
		return "[image]"
	}
	if vid := m.GetVideoMessage(); vid != nil {
		if c := vid.GetCaption(); c != "" {
			return c
		}
		return "[video]"
	}
	if doc := m.GetDocumentMessage(); doc != nil {
		if c := doc.GetCaption(); c != "" {
			return c
		}
		if fn := doc.GetFileName(); fn != "" {
			return "[document: " + fn + "]"
		}
		return "[document]"
	}
	if m.GetStickerMessage() != nil {
		return "[sticker]"
	}
	if m.GetAudioMessage() != nil {
		return "[audio]"
	}
	if m.GetContactMessage() != nil {
		return "[contact]"
	}
	if loc := m.GetLocationMessage(); loc != nil {
		return fmt.Sprintf("[location %.4f,%.4f]", loc.GetDegreesLatitude(), loc.GetDegreesLongitude())
	}
	return ""
}

func IsPrivateChat(chat types.JID) bool {
	return chat.Server != types.GroupServer && !strings.HasSuffix(chat.String(), "@g.us")
}

func IsBroadcast(chat types.JID) bool {
	return strings.HasSuffix(chat.String(), "@broadcast") || chat.Server == types.BroadcastServer
}

type InboundFilter struct {
	OwnerPhone    string
	ReplyToGroups bool
	Normalize     func(string) string
}

// InboundContext is what the handler needs after filtering.
type InboundContext struct {
	Text       string
	ConvID     string // storage key: sender phone (DM) or group JID
	Sender     string // normalized sender phone
	IsGroup    bool
	SenderName string // push name when available
}

func ShouldProcessInbound(v *events.Message, f InboundFilter) (ctx InboundContext, ok bool) {
	if v == nil || v.Info.IsFromMe {
		return InboundContext{}, false
	}
	if IsBroadcast(v.Info.Chat) {
		return InboundContext{}, false
	}

	isGroup := !IsPrivateChat(v.Info.Chat)
	if isGroup && !f.ReplyToGroups {
		return InboundContext{}, false
	}

	text := strings.TrimSpace(ExtractInboundText(v.Message))
	if text == "" {
		return InboundContext{}, false
	}

	norm := f.Normalize
	if norm == nil {
		norm = func(s string) string { return strings.TrimSpace(s) }
	}
	sender := norm(v.Info.Sender.User)
	if sender == "" {
		return InboundContext{}, false
	}
	if sender == f.OwnerPhone {
		return InboundContext{}, false
	}

	convID := sender
	if isGroup {
		convID = v.Info.Chat.String()
		text = fmt.Sprintf("[%s] %s", senderLabel(v), text)
	}

	return InboundContext{
		Text:       text,
		ConvID:     convID,
		Sender:     sender,
		IsGroup:    isGroup,
		SenderName: senderLabel(v),
	}, true
}

func senderLabel(v *events.Message) string {
	if v.Info.PushName != "" {
		return v.Info.PushName
	}
	return v.Info.Sender.User
}
