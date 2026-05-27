package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type BookingRequest struct {
	ID             string
	OwnerConv      string
	GuestPhone     string
	GuestName      string
	Status         string
	GuestSlotsJSON string
	ProposedSlot   string
	EventID        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (d *DB) InsertBookingRequest(r BookingRequest) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	_, err := d.db.Exec(
		`INSERT INTO booking_requests (id, owner_conv, guest_phone, guest_name, status, guest_slots_json, proposed_slot, event_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`,
		r.ID, r.OwnerConv, r.GuestPhone, r.GuestName, r.Status, r.GuestSlotsJSON, r.ProposedSlot, r.EventID,
	)
	return err
}

func (d *DB) GetBookingRequest(id string) (*BookingRequest, error) {
	row := d.db.QueryRow(
		`SELECT id, owner_conv, guest_phone, guest_name, status, guest_slots_json, proposed_slot, event_id, created_at, updated_at
		 FROM booking_requests WHERE id = ?`, id,
	)
	var r BookingRequest
	var created, updated string
	if err := row.Scan(&r.ID, &r.OwnerConv, &r.GuestPhone, &r.GuestName, &r.Status, &r.GuestSlotsJSON, &r.ProposedSlot, &r.EventID, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if t, err := parseSQLiteTime(created); err == nil {
		r.CreatedAt = t
	}
	if t, err := parseSQLiteTime(updated); err == nil {
		r.UpdatedAt = t
	}
	return &r, nil
}

func (d *DB) UpdateBookingRequestStatus(id, status, proposedSlot, eventID string) error {
	_, err := d.db.Exec(
		`UPDATE booking_requests SET status = ?, proposed_slot = COALESCE(NULLIF(?, ''), proposed_slot),
		 event_id = COALESCE(NULLIF(?, ''), event_id), updated_at = datetime('now') WHERE id = ?`,
		status, proposedSlot, eventID, id,
	)
	return err
}
