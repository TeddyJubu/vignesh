package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestIsAllowedAndBlocked(t *testing.T) {
	if !isAllowed("8801", nil) {
		t.Fatal("empty allowlist should allow all")
	}
	if isAllowed("8801", []string{"8802"}) {
		t.Fatal("should block when not on allowlist")
	}
	if !isAllowed("8802", []string{"8802"}) {
		t.Fatal("should allow listed number")
	}
	if !isBlocked("8809", []string{"8809"}) {
		t.Fatal("blocked number")
	}
}

func TestShouldProcessInbound_Blocklist(t *testing.T) {
	v := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   types.NewJID("8801111", types.DefaultUserServer),
				Sender: types.NewJID("8801111", types.DefaultUserServer),
			},
		},
		Message: &waE2E.Message{Conversation: proto.String("hi")},
	}
	_, ok := ShouldProcessInbound(v, InboundFilter{
		OwnerPhone:     "8809999",
		BlockedNumbers: []string{"8801111"},
		Normalize:      func(s string) string { return s },
	})
	if ok {
		t.Fatal("expected blocklist skip")
	}
}
