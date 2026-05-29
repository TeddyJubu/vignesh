package receptionist

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"ai-receptionist/internal/config"
	"ai-receptionist/internal/intent"
	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"

	"go.mau.fi/whatsmeow/types/events"
)

var phoneRE = regexp.MustCompile(`\+?\d{8,15}`)

// DispatchDay3Intent queues async jobs for Day 3 intents when sender is owner.
// Returns (handled, userFacingAck).
func (h *Handler) DispatchDay3Intent(ctx context.Context, v *events.Message, convID string, in whatsapp.InboundContext, r intent.Result, text string) (handled bool, ack string) {
	if !canPauseSender(in, h.cfg.OwnerNumber) {
		return false, ""
	}

	switch r.Intent {
	case "lead_scrape":
		if !h.cfg.Capabilities.LeadScrape {
			return true, "Lead scrape is disabled — enable capabilities.lead_scrape in config."
		}
		query, count, vertical := parseLeadScrapeRequest(text)
		payload, _ := json.Marshal(map[string]any{
			"query":    query,
			"count":    count,
			"vertical": vertical,
			"source":   text,
		})
		jobID, err := h.store.InsertAsyncJob(store.AsyncJob{
			ConvID:      convID,
			JobType:     "scrape_leads",
			Payload:     string(payload),
			NotifyOwner: true,
		})
		if err != nil {
			return true, "Could not queue lead scrape — check logs."
		}
		return true, fmt.Sprintf("Queued lead scrape (job %s) for ~%d leads. I'll email the CSV and ping you here when ready.", jobID[:8], count)

	case "research_request":
		if !h.cfg.Capabilities.MarketingResearch {
			return true, "Research is disabled — enable capabilities.marketing_research in config."
		}
		payload, _ := json.Marshal(map[string]string{"query": strings.TrimSpace(text)})
		jobID, err := h.store.InsertAsyncJob(store.AsyncJob{
			ConvID:      convID,
			JobType:     "research_marketing",
			Payload:     string(payload),
			NotifyOwner: true,
		})
		if err != nil {
			return true, "Could not queue research — check logs."
		}
		return true, fmt.Sprintf("Research queued (job %s). I'll send the report here when ready.", jobID[:8])

	case "outbound_book":
		if !h.cfg.Capabilities.OutboundBooking {
			return true, "Outbound booking is disabled — enable capabilities.outbound_booking in config."
		}
		name, phone, purpose := parseOutboundBookRequest(text)
		if phone == "" {
			return true, "Need a phone number — e.g. \"Book a meeting with John Tan, +6598765432, about partnership\""
		}
		payload, _ := json.Marshal(map[string]string{
			"contact_name":    name,
			"wa_number":       phone,
			"meeting_purpose": purpose,
			"owner_conv":      convID,
		})
		jobID, err := h.store.InsertAsyncJob(store.AsyncJob{
			ConvID:      convID,
			JobType:     "outbound_book",
			Payload:     string(payload),
			NotifyOwner: true,
		})
		if err != nil {
			return true, "Could not queue outbound booking."
		}
		return true, fmt.Sprintf("Outbound booking queued (job %s). I'll message %s with slot options.", jobID[:8], phone)

	case "group_manage":
		if !h.cfg.Capabilities.GroupAdmin {
			return true, "Group admin is disabled — enable capabilities.group_admin in config."
		}
		if handled, msg := h.handleGroupManageNL(ctx, v, in, text); handled {
			return true, msg
		}
		return true, "I couldn't parse that group command. Try: create group Epicware VIP and add +6591234567"

	default:
		return false, ""
	}
}

func parseLeadScrapeRequest(text string) (query string, count int, vertical string) {
	count = 10
	lower := strings.ToLower(text)
	if m := regexp.MustCompile(`scrape\s+(\d+)`).FindStringSubmatch(lower); len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			count = n
		}
	}
	query = strings.TrimSpace(text)
	vertical = query
	return query, count, vertical
}

func parseOutboundBookRequest(text string) (name, phone, purpose string) {
	phones := phoneRE.FindAllString(text, -1)
	if len(phones) > 0 {
		phone = config.NormalizePhone(phones[len(phones)-1])
	}
	lower := strings.ToLower(text)
	if idx := strings.Index(lower, " about "); idx >= 0 {
		purpose = strings.TrimSpace(text[idx+7:])
	}
	if idx := strings.Index(lower, " with "); idx >= 0 {
		rest := strings.TrimSpace(text[idx+6:])
		if pidx := strings.Index(strings.ToLower(rest), " about "); pidx >= 0 {
			rest = strings.TrimSpace(rest[:pidx])
		}
		// strip phone tokens
		for _, p := range phones {
			rest = strings.ReplaceAll(rest, p, "")
		}
		rest = strings.Trim(rest, " ,")
		if rest != "" {
			name = rest
		}
	}
	if purpose == "" {
		purpose = "meeting"
	}
	return name, phone, purpose
}
