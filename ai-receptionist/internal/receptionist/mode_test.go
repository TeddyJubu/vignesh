package receptionist

import (
	"testing"

	"ai-receptionist/internal/store"
	"ai-receptionist/internal/whatsapp"
)

func TestResolveMode(t *testing.T) {
	in := whatsapp.InboundContext{IsGroup: false}
	if got := ResolveMode(nil, in, "hello"); got != modeSales {
		t.Fatalf("dm default=%q", got)
	}
	in.IsGroup = true
	if got := ResolveMode(nil, in, "hello"); got != modeCS {
		t.Fatalf("group=%q", got)
	}
	if got := ResolveMode(nil, in, "book an appointment"); got != modeBooking {
		t.Fatalf("booking=%q", got)
	}
	c := &store.Contact{Mode: modeCS}
	if got := ResolveMode(c, in, "book"); got != modeCS {
		t.Fatalf("stored mode=%q", got)
	}
}
