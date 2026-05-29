package receptionist

import (
	"errors"
	"strings"
	"testing"
)

func TestCustomerFailureMessage_parseJSON(t *testing.T) {
	msg := customerFailureMessage("anthropic", errors.New(`parse AI JSON: json: cannot unmarshal bool`))
	if strings.Contains(msg, "terminal") || strings.Contains(msg, "couldn't reach") {
		t.Fatalf("generic provider message leaked: %q", msg)
	}
	if !strings.Contains(msg, "formatting") {
		t.Fatalf("expected formatting hint: %q", msg)
	}
}

func TestCustomerFailureMessage_timeout(t *testing.T) {
	msg := customerFailureMessage("anthropic", errors.New(`Post "https://api.anthropic.com/v1/messages": context deadline exceeded`))
	if !strings.Contains(msg, "too long") {
		t.Fatalf("expected timeout hint: %q", msg)
	}
}

func TestCustomerFailureMessage_model404(t *testing.T) {
	msg := customerFailureMessage("anthropic", errors.New(`Anthropic HTTP 404: model: claude-3-5-sonnet-latest`))
	if !strings.Contains(msg, "model") {
		t.Fatalf("expected model hint: %q", msg)
	}
}
