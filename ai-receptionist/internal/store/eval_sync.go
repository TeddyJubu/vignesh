package store

import "fmt"

// RefreshEmbeddedAgentNotes updates identity_soul, client_instructions, and mode runbooks
// from embedded knowledge files. Used by juliaeval so local runs always match the repo.
func (d *DB) RefreshEmbeddedAgentNotes() error {
	notes := map[string]string{
		"identity_soul":        defaultIdentitySoul(),
		"client_instructions": defaultClientInstructions(),
		"julia-cs":             RunbookCS,
		"julia-sales":          RunbookSales,
		"julia-booking":        RunbookBooking,
	}
	for key, content := range notes {
		if err := d.UpsertAgentNote(key, content); err != nil {
			return fmt.Errorf("refresh agent_note %s: %w", key, err)
		}
	}
	return nil
}
