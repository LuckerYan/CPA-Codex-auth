package auth

import (
	"strings"
	"testing"
	"time"
)

func TestCodexTokenInvalidatedDetection(t *testing.T) {
	const message = "401 Your authentication token has been invalidated. Please try signing in again."
	if !IsAuthenticationTokenInvalidated(401, message) {
		t.Fatalf("expected exact invalidated-token 401 message to be detected")
	}
	if !IsAuthenticationTokenInvalidated(0, message) {
		t.Fatalf("expected embedded 401 invalidated-token message to be detected")
	}
	if IsAuthenticationTokenInvalidated(403, message) {
		t.Fatalf("did not expect a non-401 status to be detected")
	}
	if IsAuthenticationTokenInvalidated(401, "401 unauthorized") {
		t.Fatalf("did not expect a generic 401 to be treated as token invalidation")
	}
}

func TestMarkAuthBannedPersistsAccountStatus(t *testing.T) {
	auth := &Auth{
		ID:       "codex.json",
		Provider: "codex",
		Status:   StatusActive,
		Metadata: map[string]any{},
	}
	const message = "401 Your authentication token has been invalidated. Please try signing in again."

	MarkAuthBanned(auth, message, time.Unix(123, 0))

	if !auth.IsBanned() {
		t.Fatalf("expected auth to be banned")
	}
	if !auth.IsBlocked() {
		t.Fatalf("expected banned auth to be blocked")
	}
	if auth.Disabled {
		t.Fatalf("banned status should not toggle the operator disabled flag")
	}
	if got := AccountStatusString(auth); got != string(StatusBanned) {
		t.Fatalf("account status = %q, want %q", got, StatusBanned)
	}
	if PersistedAccountStatus(auth.Metadata) != StatusBanned {
		t.Fatalf("expected persisted metadata status to be banned")
	}
	if got := PersistedAccountStatusMessage(auth.Metadata); !strings.Contains(got, "invalidated") {
		t.Fatalf("persisted status message = %q, want invalidated text", got)
	}

	reloaded := &Auth{
		ID:       auth.ID,
		Provider: auth.Provider,
		Status:   StatusActive,
		Metadata: auth.Metadata,
	}
	ApplyPersistedAccountStatus(reloaded)
	if !reloaded.IsBanned() || !reloaded.IsBlocked() {
		t.Fatalf("expected persisted banned status to survive reload")
	}
}
