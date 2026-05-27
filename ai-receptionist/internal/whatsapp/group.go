package whatsapp

import (
	"context"
	"fmt"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

// CreateGroup creates a WhatsApp group with the given subject and participant phone numbers.
func CreateGroup(ctx context.Context, wa *Client, name string, participantPhones []string) (*types.GroupInfo, error) {
	if wa == nil || wa.WM == nil {
		return nil, fmt.Errorf("whatsapp client not ready")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("group name required")
	}
	if len(name) > 25 {
		name = name[:25]
	}
	var participants []types.JID
	for _, p := range participantPhones {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		participants = append(participants, PhoneToJID(p))
	}
	return wa.WM.CreateGroup(ctx, whatsmeow.ReqCreateGroup{
		Name:         name,
		Participants: participants,
	})
}

// AddGroupParticipants adds phones to an existing group JID string.
func AddGroupParticipants(ctx context.Context, wa *Client, groupJID string, phones []string) error {
	if wa == nil || wa.WM == nil {
		return fmt.Errorf("whatsapp client not ready")
	}
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return err
	}
	var participants []types.JID
	for _, p := range phones {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		participants = append(participants, PhoneToJID(p))
	}
	_, err = wa.WM.UpdateGroupParticipants(ctx, jid, participants, whatsmeow.ParticipantChangeAdd)
	return err
}

// SetGroupName renames a group.
func SetGroupName(ctx context.Context, wa *Client, groupJID, name string) error {
	if wa == nil || wa.WM == nil {
		return fmt.Errorf("whatsapp client not ready")
	}
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return err
	}
	return wa.WM.SetGroupName(ctx, jid, strings.TrimSpace(name))
}

// GetGroupInviteLink returns the invite link for a group.
func GetGroupInviteLink(ctx context.Context, wa *Client, groupJID string) (string, error) {
	if wa == nil || wa.WM == nil {
		return "", fmt.Errorf("whatsapp client not ready")
	}
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return "", err
	}
	return wa.WM.GetGroupInviteLink(ctx, jid, false)
}
