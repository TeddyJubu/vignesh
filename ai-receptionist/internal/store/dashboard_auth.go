package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type DashboardSession struct {
	Phone       string          `json:"phone"`
	Role        string          `json:"role"`
	Permissions map[string]bool `json:"permissions,omitempty"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

func hashStringSHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func generateOTP6() (string, error) {
	// 000000..999999
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (d *DB) CreateDashboardOTP(phone string, ttl time.Duration) (code string, expiresAt time.Time, err error) {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	code, err = generateOTP6()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = time.Now().Add(ttl).UTC()
	codeHash := hashStringSHA256Hex(phone + ":" + code)

	// Best-effort cleanup previous codes.
	_, _ = d.db.Exec(`DELETE FROM dashboard_otp_codes WHERE phone = ?`, phone)
	_, err = d.db.Exec(
		`INSERT INTO dashboard_otp_codes (phone, code_hash, expires_at, created_at)
		 VALUES (?, ?, ?, datetime('now'))`,
		phone, codeHash, expiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return "", time.Time{}, err
	}
	return code, expiresAt, nil
}

func (d *DB) VerifyDashboardOTP(phone, code string) (bool, error) {
	row := d.db.QueryRow(
		`SELECT code_hash, expires_at FROM dashboard_otp_codes WHERE phone = ?`,
		phone,
	)
	var codeHash, expiresRaw string
	if err := row.Scan(&codeHash, &expiresRaw); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	expiresAt, err := parseSQLiteTime(expiresRaw)
	if err != nil {
		return false, err
	}
	if time.Now().UTC().After(expiresAt.UTC()) {
		_, _ = d.db.Exec(`DELETE FROM dashboard_otp_codes WHERE phone = ?`, phone)
		return false, nil
	}
	want := hashStringSHA256Hex(phone + ":" + code)
	ok := subtleConstantTimeEquals(want, strings.TrimSpace(codeHash))
	if ok {
		_, _ = d.db.Exec(`DELETE FROM dashboard_otp_codes WHERE phone = ?`, phone)
	}
	return ok, nil
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (d *DB) CreateDashboardSession(phone string, ttl time.Duration) (token string, sess *DashboardSession, err error) {
	if ttl <= 0 {
		ttl = 14 * 24 * time.Hour
	}
	roleRow, err := d.GetAccessRole(phone)
	if err != nil {
		return "", nil, err
	}
	if roleRow == nil || (roleRow.Role != "admin" && roleRow.Role != "manager") {
		return "", nil, fmt.Errorf("not allowed")
	}
	token, err = generateSessionToken()
	if err != nil {
		return "", nil, err
	}
	tokenHash := hashStringSHA256Hex(token)
	expiresAt := time.Now().Add(ttl).UTC()
	perms := roleRow.Permissions
	if perms == nil {
		perms = map[string]bool{}
	}
	permsJSON, _ := json.Marshal(perms)
	_, err = d.db.Exec(
		`INSERT INTO dashboard_sessions (token_hash, phone, role, permissions_json, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, datetime('now'))`,
		tokenHash, phone, roleRow.Role, string(permsJSON), expiresAt.Format(time.RFC3339),
	)
	if err != nil {
		return "", nil, err
	}
	return token, &DashboardSession{
		Phone:       phone,
		Role:        roleRow.Role,
		Permissions: perms,
		ExpiresAt:   expiresAt,
	}, nil
}

func (d *DB) GetDashboardSessionByToken(token string) (*DashboardSession, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	tokenHash := hashStringSHA256Hex(token)
	row := d.db.QueryRow(
		`SELECT phone, role, permissions_json, expires_at
		 FROM dashboard_sessions
		 WHERE token_hash = ?`,
		tokenHash,
	)
	var phone, role, permsRaw, expiresRaw string
	if err := row.Scan(&phone, &role, &permsRaw, &expiresRaw); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	expiresAt, err := parseSQLiteTime(expiresRaw)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(expiresAt.UTC()) {
		_, _ = d.db.Exec(`DELETE FROM dashboard_sessions WHERE token_hash = ?`, tokenHash)
		return nil, nil
	}
	perms := map[string]bool{}
	if strings.TrimSpace(permsRaw) != "" && strings.TrimSpace(permsRaw) != "{}" {
		_ = json.Unmarshal([]byte(permsRaw), &perms)
	}
	if perms == nil {
		perms = map[string]bool{}
	}
	return &DashboardSession{
		Phone:       phone,
		Role:        normalizeRole(role),
		Permissions: perms,
		ExpiresAt:   expiresAt,
	}, nil
}

func (d *DB) RevokeDashboardSession(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	tokenHash := hashStringSHA256Hex(token)
	_, err := d.db.Exec(`DELETE FROM dashboard_sessions WHERE token_hash = ?`, tokenHash)
	return err
}

// subtleConstantTimeEquals duplicates the httpapi version to keep store independent.
func subtleConstantTimeEquals(a, b string) bool {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	var out byte
	for i := 0; i < n; i++ {
		var ca, cb byte
		if i < len(a) {
			ca = a[i]
		}
		if i < len(b) {
			cb = b[i]
		}
		out |= ca ^ cb
	}
	return out == 0 && len(a) == len(b)
}

