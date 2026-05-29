package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type AccessRole struct {
	Phone       string         `json:"phone"`
	Role        string         `json:"role"`
	Permissions map[string]bool `json:"permissions,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func normalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func (d *DB) GetAccessRole(phone string) (*AccessRole, error) {
	row := d.db.QueryRow(
		`SELECT phone, role, permissions_json, created_at, updated_at
		 FROM access_roles WHERE phone = ?`,
		phone,
	)
	var r AccessRole
	var permsRaw, createdRaw, updatedRaw string
	if err := row.Scan(&r.Phone, &r.Role, &permsRaw, &createdRaw, &updatedRaw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.Permissions = map[string]bool{}
	if strings.TrimSpace(permsRaw) != "" && strings.TrimSpace(permsRaw) != "{}" {
		_ = json.Unmarshal([]byte(permsRaw), &r.Permissions)
	}
	if r.Permissions == nil {
		r.Permissions = map[string]bool{}
	}
	if t, err := parseSQLiteTime(createdRaw); err == nil {
		r.CreatedAt = t
	}
	if t, err := parseSQLiteTime(updatedRaw); err == nil {
		r.UpdatedAt = t
	}
	r.Role = normalizeRole(r.Role)
	return &r, nil
}

func (d *DB) UpsertAccessRole(phone, role string, permissions map[string]bool) error {
	role = normalizeRole(role)
	if role != "admin" && role != "manager" && role != "client" {
		return fmt.Errorf("invalid role")
	}
	if permissions == nil {
		permissions = map[string]bool{}
	}
	permsJSON, _ := json.Marshal(permissions)
	_, err := d.db.Exec(
		`INSERT INTO access_roles (phone, role, permissions_json, created_at, updated_at)
		 VALUES (?, ?, ?, datetime('now'), datetime('now'))
		 ON CONFLICT(phone) DO UPDATE
		   SET role = excluded.role,
		       permissions_json = excluded.permissions_json,
		       updated_at = datetime('now')`,
		phone, role, string(permsJSON),
	)
	return err
}

func (d *DB) DeleteAccessRole(phone string) error {
	_, err := d.db.Exec(`DELETE FROM access_roles WHERE phone = ?`, phone)
	return err
}

func (d *DB) ListAccessRoles(role string, limit int) ([]AccessRole, error) {
	role = normalizeRole(role)
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}
	var (
		rows *sql.Rows
		err  error
	)
	if role == "" {
		rows, err = d.db.Query(
			`SELECT phone, role, permissions_json, created_at, updated_at
			 FROM access_roles
			 ORDER BY role ASC, phone ASC
			 LIMIT ?`,
			limit,
		)
	} else {
		rows, err = d.db.Query(
			`SELECT phone, role, permissions_json, created_at, updated_at
			 FROM access_roles
			 WHERE role = ?
			 ORDER BY phone ASC
			 LIMIT ?`,
			role, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AccessRole{}
	for rows.Next() {
		var r AccessRole
		var permsRaw, createdRaw, updatedRaw string
		if err := rows.Scan(&r.Phone, &r.Role, &permsRaw, &createdRaw, &updatedRaw); err != nil {
			return nil, err
		}
		r.Role = normalizeRole(r.Role)
		r.Permissions = map[string]bool{}
		if strings.TrimSpace(permsRaw) != "" && strings.TrimSpace(permsRaw) != "{}" {
			_ = json.Unmarshal([]byte(permsRaw), &r.Permissions)
		}
		if r.Permissions == nil {
			r.Permissions = map[string]bool{}
		}
		if t, err := parseSQLiteTime(createdRaw); err == nil {
			r.CreatedAt = t
		}
		if t, err := parseSQLiteTime(updatedRaw); err == nil {
			r.UpdatedAt = t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// EnsureClientRole upserts a client role only if the number
// isn't already an admin or manager.
func (d *DB) EnsureClientRole(phone string) error {
	r, err := d.GetAccessRole(phone)
	if err != nil {
		return err
	}
	if r != nil && (r.Role == "admin" || r.Role == "manager") {
		return nil
	}
	if r != nil && r.Role == "client" {
		return nil
	}
	return d.UpsertAccessRole(phone, "client", nil)
}

