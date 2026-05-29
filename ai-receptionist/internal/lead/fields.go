package lead

import "strings"

// Required fields for qualification.
var Required = []string{
	"name",
	"email",
	"business_type",
	"service_needed",
	"budget",
	"timeline",
	"current_website",
}

// Optional fields stored when provided.
var Optional = []string{"best_time"}

func Merge(existing map[string]string, updates map[string]string) map[string]string {
	out := make(map[string]string, len(existing)+len(updates))
	for k, v := range existing {
		if strings.TrimSpace(v) != "" {
			out[k] = v
		}
	}
	for k, v := range updates {
		v = strings.TrimSpace(v)
		if v != "" {
			out[k] = v
		}
	}
	return out
}

func Missing(data map[string]string) []string {
	var miss []string
	for _, f := range Required {
		if strings.TrimSpace(data[f]) == "" {
			miss = append(miss, f)
		}
	}
	return miss
}

func IsQualified(data map[string]string) bool {
	return len(Missing(data)) == 0
}

func DenormalizedName(data map[string]string) string {
	return strings.TrimSpace(data["name"])
}
