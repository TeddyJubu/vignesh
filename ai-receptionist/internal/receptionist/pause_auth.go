package receptionist

import (
	"ai-receptionist/internal/config"
	"ai-receptionist/internal/whatsapp"
)

// canPauseSender limits takeover commands to the configured owner number.
func canPauseSender(in whatsapp.InboundContext, ownerNumber string) bool {
	return config.NormalizePhone(in.Sender) == config.NormalizePhone(ownerNumber)
}
