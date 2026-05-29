package receptionist

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

var groupPhoneRE = regexp.MustCompile(`\+?\d{8,15}`)

type groupNLAction struct {
	Kind     string // create, add, remove, rename, announce, invite
	GroupRef string
	Name     string
	Phones   []string
	Message  string
}

// IsGroupManageNL detects natural-language group management (owner).
func IsGroupManageNL(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if strings.Contains(lower, "group") {
		return strings.Contains(lower, "create") ||
			strings.Contains(lower, "add") ||
			strings.Contains(lower, "remove") ||
			strings.Contains(lower, "rename") ||
			strings.Contains(lower, "announce") ||
			strings.Contains(lower, "invite")
	}
	return IsGroupAdminKeyword(text)
}

func parseGroupNL(text string) (groupNLAction, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	a := groupNLAction{}

	// create group X and add phones
	if strings.Contains(lower, "create") && strings.Contains(lower, "group") {
		a.Kind = "create"
		name := extractQuotedOrNamed(text, "group")
		if name == "" {
			if idx := strings.Index(lower, "called "); idx >= 0 {
				rest := strings.TrimSpace(text[idx+7:])
				if end := strings.IndexAny(rest, " and,\n"); end >= 0 {
					name = strings.TrimSpace(rest[:end])
				} else {
					name = rest
				}
			}
		}
		a.Name = strings.Trim(name, `"'" `)
		a.Phones = extractPhones(text)
		if a.Name != "" {
			return a, true
		}
	}

	if strings.Contains(lower, "add") && strings.Contains(lower, "group") {
		a.Kind = "add"
		a.Phones = extractPhones(text)
		a.GroupRef = extractGroupJID(text)
		if len(a.Phones) > 0 && a.GroupRef != "" {
			return a, true
		}
	}

	if strings.Contains(lower, "remove") && strings.Contains(lower, "group") {
		a.Kind = "remove"
		a.Phones = extractPhones(text)
		a.GroupRef = extractGroupJID(text)
		if len(a.Phones) > 0 && a.GroupRef != "" {
			return a, true
		}
	}

	if strings.Contains(lower, "rename") && strings.Contains(lower, "group") {
		a.Kind = "rename"
		a.GroupRef = extractGroupJID(text)
		if idx := strings.Index(lower, " to "); idx >= 0 {
			a.Name = strings.Trim(strings.TrimSpace(text[idx+4:]), `"`)
		}
		if a.GroupRef != "" && a.Name != "" {
			return a, true
		}
	}

	if strings.Contains(lower, "announce") {
		a.Kind = "announce"
		a.GroupRef = extractGroupJID(text)
		if idx := strings.Index(lower, "announce"); idx >= 0 {
			a.Message = strings.TrimSpace(text[idx+len("announce"):])
			a.Message = strings.TrimPrefix(a.Message, " to group ")
			a.Message = strings.TrimSpace(a.Message)
		}
		if a.GroupRef != "" && a.Message != "" {
			return a, true
		}
	}

	return a, false
}

func (h *Handler) handleGroupManageNL(ctx context.Context, v *events.Message, in whatsapp.InboundContext, text string) (handled bool, msg string) {
	if IsGroupAdminKeyword(text) {
		h.handleGroupAdmin(ctx, v, in)
		return true, ""
	}
	action, ok := parseGroupNL(text)
	if !ok {
		return false, ""
	}
	switch action.Kind {
	case "create":
		info, err := whatsapp.CreateGroup(ctx, h.wa, action.Name, action.Phones)
		if err != nil {
			return true, "Could not create group: " + err.Error()
		}
		link, _ := whatsapp.GetGroupInviteLink(ctx, h.wa, info.JID.String())
		out := fmt.Sprintf("Created group %q.", info.Name)
		if link != "" {
			out += "\nInvite: " + link
		}
		return true, out
	case "add":
		if err := whatsapp.AddGroupParticipants(ctx, h.wa, action.GroupRef, action.Phones); err != nil {
			return true, "Could not add participants: " + err.Error()
		}
		return true, "Added participants to group."
	case "remove":
		if err := whatsapp.RemoveGroupParticipants(ctx, h.wa, action.GroupRef, action.Phones); err != nil {
			return true, "Could not remove participants: " + err.Error()
		}
		return true, "Removed participants from group."
	case "rename":
		if err := whatsapp.SetGroupName(ctx, h.wa, action.GroupRef, action.Name); err != nil {
			return true, "Could not rename group: " + err.Error()
		}
		return true, "Group renamed."
	case "announce":
		if err := whatsapp.SendGroupText(ctx, h.wa, action.GroupRef, action.Message); err != nil {
			return true, "Could not announce: " + err.Error()
		}
		return true, "Announcement sent."
	}
	return false, ""
}

func extractPhones(text string) []string {
	var out []string
	for _, m := range groupPhoneRE.FindAllString(text, -1) {
		if n := config.NormalizePhone(m); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func extractGroupJID(text string) string {
	for _, tok := range strings.Fields(text) {
		if strings.Contains(tok, "@g.us") {
			return tok
		}
	}
	return ""
}

func extractQuotedOrNamed(text, after string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, after)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(text[idx+len(after):])
	rest = strings.TrimPrefix(rest, "called")
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, `"`) {
		if end := strings.Index(rest[1:], `"`); end >= 0 {
			return rest[1 : end+1]
		}
	}
	if strings.HasPrefix(rest, "'") {
		if end := strings.Index(rest[1:], "'"); end >= 0 {
			return rest[1 : end+1]
		}
	}
	if rest != "" {
		return trimNameAtSeparator(rest)
	}
	return ""
}

func trimNameAtSeparator(rest string) string {
	rest = strings.TrimSpace(rest)
	lower := strings.ToLower(rest)
	for _, sep := range []string{" and add ", " and invite ", " with ", " and ", ","} {
		if idx := strings.Index(lower, sep); idx > 0 {
			return strings.TrimSpace(rest[:idx])
		}
	}
	// strip trailing phone if present
	if idx := strings.Index(lower, "+"); idx > 0 {
		return strings.TrimSpace(rest[:idx])
	}
	return rest
}
