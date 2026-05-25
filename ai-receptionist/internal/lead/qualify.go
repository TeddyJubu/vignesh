package lead

import (
	"fmt"
	"strings"
)

func AdminSummary(businessName, phone string, data map[string]string, aiSummary string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "🔥 New qualified lead — %s\n", businessName)
	fmt.Fprintf(&b, "Name: %s\n", data["name"])
	fmt.Fprintf(&b, "Phone: %s\n", phone)
	fmt.Fprintf(&b, "Business: %s\n", data["business_type"])
	fmt.Fprintf(&b, "Needs: %s\n", data["service_needed"])
	fmt.Fprintf(&b, "Budget: %s\n", data["budget"])
	fmt.Fprintf(&b, "Timeline: %s\n", data["timeline"])
	fmt.Fprintf(&b, "Website: %s\n", data["current_website"])
	if bt := strings.TrimSpace(data["best_time"]); bt != "" {
		fmt.Fprintf(&b, "Best time: %s\n", bt)
	}
	if s := strings.TrimSpace(aiSummary); s != "" {
		fmt.Fprintf(&b, "\nSummary:\n%s", s)
	}
	return strings.TrimSpace(b.String())
}
