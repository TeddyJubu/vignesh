package composio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveConnectedAccountExplicit(t *testing.T) {
	c := New("test-key")
	id, err := c.ResolveConnectedAccount(context.Background(), "googlecalendar", "acct-explicit")
	if err != nil {
		t.Fatal(err)
	}
	if id != "acct-explicit" {
		t.Fatalf("got %q want acct-explicit", id)
	}
}

func TestListConnectedAccountsAndResolve(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3.1/connected_accounts" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q", got)
		}
		if !strings.Contains(r.URL.RawQuery, "toolkit_slugs=googlecalendar") {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":      "cal-123",
					"user_id": "default",
					"status":  "ACTIVE",
					"toolkit": map[string]string{"slug": "googlecalendar"},
				},
			},
		})
	}))
	defer srv.Close()

	c := New("test-key")
	c.baseURL = strings.TrimSuffix(srv.URL, "") + "/api/v3.1"

	accounts, err := c.ListConnectedAccounts(context.Background(), "googlecalendar", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 1 || accounts[0].ID != "cal-123" {
		t.Fatalf("accounts = %+v", accounts)
	}

	id, err := c.ResolveConnectedAccount(context.Background(), "googlecalendar", "")
	if err != nil {
		t.Fatal(err)
	}
	if id != "cal-123" {
		t.Fatalf("resolved id = %q", id)
	}
}

func TestExecuteTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/tools/execute/GMAIL_SEND_EMAIL") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["connected_account_id"] != "gmail-1" {
			t.Fatalf("connected_account_id = %v", body["connected_account_id"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"successful": true,
			"data":       map[string]any{"message_id": "msg-1"},
		})
	}))
	defer srv.Close()

	c := New("test-key")
	c.baseURL = strings.TrimSuffix(srv.URL, "") + "/api/v3.1"

	res, err := c.Execute(context.Background(), "GMAIL_SEND_EMAIL", ExecuteRequest{
		ConnectedAccountID: "gmail-1",
		UserID:             "default",
		Text:               "Send test email",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Successful {
		t.Fatalf("successful=false err=%q", res.Error)
	}
}

func TestNormalizeToolkitSlug(t *testing.T) {
	cases := map[string]string{
		"GOOGLECALENDAR": "googlecalendar",
		"google_calendar": "googlecalendar",
		"gmail":           "gmail",
	}
	for in, want := range cases {
		if got := normalizeToolkitSlug(in); got != want {
			t.Errorf("normalizeToolkitSlug(%q) = %q want %q", in, got, want)
		}
	}
}
