package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestGroupAllowlistAndMention(t *testing.T) {
	groupJID := types.NewJID("120363000000000000", types.GroupServer)
	v := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   groupJID,
				Sender: types.NewJID("8801111", types.DefaultUserServer),
			},
		},
		Message: &waE2E.Message{Conversation: proto.String("hello julia")},
	}
	_, ok := ShouldProcessInbound(v, InboundFilter{
		OwnerPhone:          "8809999",
		ReplyToGroups:       true,
		SupportGroupJIDs:    []string{groupJID.String()},
		GroupReplyPolicy:    "mention_or_owner",
		GroupMentionAliases: []string{"julia"},
		Normalize:           func(s string) string { return s },
	})
	if !ok {
		t.Fatal("expected group message with julia mention")
	}

	v2 := *v
	v2.Message = &waE2E.Message{Conversation: proto.String("random chat")}
	_, ok2 := ShouldProcessInbound(&v2, InboundFilter{
		OwnerPhone:          "8809999",
		ReplyToGroups:       true,
		SupportGroupJIDs:    []string{groupJID.String()},
		GroupReplyPolicy:    "mention_or_owner",
		GroupMentionAliases: []string{"julia"},
		Normalize:           func(s string) string { return s },
	})
	if ok2 {
		t.Fatal("expected skip without mention")
	}

	v3 := *v
	v3.Message = &waE2E.Message{Conversation: proto.String("talking about julian")}
	_, ok3 := ShouldProcessInbound(&v3, InboundFilter{
		OwnerPhone:          "8809999",
		ReplyToGroups:       true,
		SupportGroupJIDs:    []string{groupJID.String()},
		GroupReplyPolicy:    "mention_or_owner",
		GroupMentionAliases: []string{"julia"},
		Normalize:           func(s string) string { return s },
	})
	if ok3 {
		t.Fatal("expected skip when alias is only a substring")
	}
}
