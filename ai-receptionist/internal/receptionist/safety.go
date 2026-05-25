package receptionist

import (
	"regexp"
	"strings"
)

const deferReply = "Let me pass this to the team so they can confirm properly."

var (
	priceGuarantee = regexp.MustCompile(`(?i)(guarantee|guaranteed|fixed price|exact(?:ly)?\s+\$|only\s+\$\d|price is\s+\$|will cost exactly)`)
	bookingConfirm = regexp.MustCompile(`(?i)(booked|booking confirmed|appointment (?:is )?confirmed|scheduled for (?:monday|tuesday|wednesday|thursday|friday|saturday|sunday|\d)|see you (?:on|at) \d)`)
)

func SanitizeReply(reply string) string {
	r := strings.TrimSpace(reply)
	if r == "" {
		return deferReply
	}
	if priceGuarantee.MatchString(r) || bookingConfirm.MatchString(r) {
		return deferReply
	}
	return r
}
