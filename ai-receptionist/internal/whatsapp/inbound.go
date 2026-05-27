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
	OwnerPhone          string
	ReplyToGroups       bool
	ReplyToSelf         bool
	OwnJID              types.JID
	Sent                *OutboundTracker
	Normalize           func(string) string
	AllowedNumbers      []string // if non-empty, only these senders are processed
	BlockedNumbers      []string
	SupportGroupJIDs    []string // if non-empty, only these group JIDs
	GroupReplyPolicy    string   // always | mention_or_owner | owner_only
	GroupMentionAliases []string // e.g. julia
}

// IsSelfChat detects WhatsApp "Message yourself" (notes-to-self).
// Chat and Sender are the same JID; own JID match is a fallback when linked as PN/LID mix.
func IsSelfChat(v *events.Message, own types.JID) bool {
	if v == nil || !IsPrivateChat(v.Info.Chat) {
		return false
	}
	chat, sender := v.Info.Chat, v.Info.Sender
	if chat.User != "" && chat.User == sender.User && chat.Server == sender.Server {
		return true
	}
	if !own.IsEmpty() {
		if chat.User == own.User && chat.Server == own.Server {
			return true
		}
		if sender.User == own.User && sender.Server == own.Server && v.Info.IsFromMe {
			return true
		}
	}
	return false
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
	if v == nil {
		return InboundContext{}, false
	}

	selfChat := IsSelfChat(v, f.OwnJID)

	if v.Info.IsFromMe {
		if !f.ReplyToSelf || !selfChat {
			return InboundContext{}, false
		}
		if f.Sent != nil && f.Sent.IsOurs(v.Info.ID) {
			return InboundContext{}, false
		}
	}

	if IsBroadcast(v.Info.Chat) {
		return InboundContext{}, false
	}

	isGroup := !IsPrivateChat(v.Info.Chat)
	if isGroup && !f.ReplyToGroups {
		return InboundContext{}, false
	}
	if isGroup && len(f.SupportGroupJIDs) > 0 && !isGroupAllowlisted(v.Info.Chat.String(), f.SupportGroupJIDs) {
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
	if sender == "" && selfChat {
		sender = norm(f.OwnJID.User)
	}
	if sender == "" {
		return InboundContext{}, false
	}
	if sender == f.OwnerPhone && !selfChat {
		return InboundContext{}, false
	}

	if isBlocked(sender, f.BlockedNumbers) {
		return InboundContext{}, false
	}
	if !isAllowed(sender, f.AllowedNumbers) {
		return InboundContext{}, false
	}

	if isGroup && !groupReplyAllowed(v, f, sender, text) {
		return InboundContext{}, false
	}

	convID := sender
	if selfChat {
		convID = "self:" + sender
	} else if isGroup {
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

func isBlocked(sender string, blocked []string) bool {
	for _, n := range blocked {
		if n == sender {
			return true
		}
	}
	return false
}

func isAllowed(sender string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, n := range allowed {
		if n == sender {
			return true
		}
	}
	return false
}

func senderLabel(v *events.Message) string {
	if v.Info.PushName != "" {
		return v.Info.PushName
	}
	return v.Info.Sender.User
}

func isGroupAllowlisted(groupJID string, allowlist []string) bool {
	g := strings.TrimSpace(groupJID)
	for _, id := range allowlist {
		if strings.TrimSpace(id) == g {
			return true
		}
	}
	return false
}

func groupReplyAllowed(v *events.Message, f InboundFilter, sender, text string) bool {
	policy := strings.ToLower(strings.TrimSpace(f.GroupReplyPolicy))
	if policy == "" {
		policy = "mention_or_owner"
	}
	switch policy {
	case "always":
		return true
	case "owner_only":
		return sender == f.OwnerPhone
	case "mention_or_owner":
		if sender == f.OwnerPhone {
			return true
		}
		return messageMentionsBot(v, f.OwnJID, f.GroupMentionAliases, text)
	default:
		return messageMentionsBot(v, f.OwnJID, f.GroupMentionAliases, text)
	}
}

func messageMentionsBot(v *events.Message, own types.JID, aliases []string, text string) bool {
	if v == nil || v.Message == nil {
		return false
	}
	mentioned := false
	checkJID := func(jid string) {
		if jid == "" || own.IsEmpty() {
			return
		}
		if mj, err := types.ParseJID(jid); err == nil &&
			mj.User == own.User && mj.Server == own.Server {
			mentioned = true
		}
	}
	if et := v.Message.GetExtendedTextMessage(); et != nil {
		if ci := et.GetContextInfo(); ci != nil {
			for _, m := range ci.GetMentionedJID() {
				checkJID(m)
			}
		}
	}
	if mentioned {
		return true
	}
	lower := strings.ToLower(text)
	for _, a := range aliases {
		a = strings.TrimSpace(strings.ToLower(a))
		if a == "" {
			continue
		}
		if strings.Contains(lower, "@"+a) || containsToken(lower, a) {
			return true
		}
	}
	return false
}

func containsToken(text, token string) bool {
	for _, f := range strings.FieldsFunc(text, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_')
	}) {
		if f == token {
			return true
		}
	}
	return false
}
