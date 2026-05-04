package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	persistedAccountStatusKey      = "cli_proxy_account_status"
	persistedStatusMessageKey      = "cli_proxy_status_message"
	persistedAccountStatusBanned   = "banned"
	persistedAccountStatusDisabled = "disabled"
)

// PersistedAccountStatus returns the persisted account status encoded in metadata.
// Missing or unknown values default to active.
func PersistedAccountStatus(metadata map[string]any) Status {
	if len(metadata) == 0 {
		return StatusActive
	}
	raw, ok := metadata[persistedAccountStatusKey]
	if !ok {
		return StatusActive
	}
	switch normalizePersistedAccountStatus(raw) {
	case persistedAccountStatusBanned:
		return StatusBanned
	case persistedAccountStatusDisabled:
		return StatusDisabled
	default:
		return StatusActive
	}
}

// PersistedAccountStatusMessage returns the persisted status message encoded in metadata.
func PersistedAccountStatusMessage(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	if raw, ok := metadata[persistedStatusMessageKey]; ok {
		if msg, okMsg := raw.(string); okMsg {
			return strings.TrimSpace(msg)
		}
	}
	return ""
}

// ApplyPersistedAccountStatus updates the runtime auth state from persisted metadata.
func ApplyPersistedAccountStatus(auth *Auth) {
	if auth == nil {
		return
	}
	if disabled, _ := auth.Metadata["disabled"].(bool); disabled {
		auth.Disabled = true
	}
	switch PersistedAccountStatus(auth.Metadata) {
	case StatusBanned:
		auth.Status = StatusBanned
		auth.Unavailable = true
		if msg := PersistedAccountStatusMessage(auth.Metadata); msg != "" {
			auth.StatusMessage = msg
		} else if strings.TrimSpace(auth.StatusMessage) == "" {
			auth.StatusMessage = "token invalidated"
		}
	case StatusDisabled:
		auth.Status = StatusDisabled
		auth.Disabled = true
		if msg := PersistedAccountStatusMessage(auth.Metadata); msg != "" {
			auth.StatusMessage = msg
		}
	default:
		if auth.Disabled {
			auth.Status = StatusDisabled
		} else if auth.Status == "" || auth.Status == StatusUnknown {
			auth.Status = StatusActive
		}
	}
}

// SetPersistedAccountStatus stores the account status and optional status message in metadata.
func SetPersistedAccountStatus(auth *Auth, status Status, message string) {
	if auth == nil || auth.Metadata == nil {
		return
	}
	switch status {
	case StatusBanned:
		auth.Metadata[persistedAccountStatusKey] = persistedAccountStatusBanned
		if trimmed := strings.TrimSpace(message); trimmed != "" {
			auth.Metadata[persistedStatusMessageKey] = trimmed
		} else {
			delete(auth.Metadata, persistedStatusMessageKey)
		}
	case StatusDisabled:
		auth.Metadata[persistedAccountStatusKey] = persistedAccountStatusDisabled
		if trimmed := strings.TrimSpace(message); trimmed != "" {
			auth.Metadata[persistedStatusMessageKey] = trimmed
		} else {
			delete(auth.Metadata, persistedStatusMessageKey)
		}
	default:
		delete(auth.Metadata, persistedAccountStatusKey)
		delete(auth.Metadata, persistedStatusMessageKey)
	}
}

// SyncPersistedAccountStatus mirrors the runtime status into persistent metadata.
func SyncPersistedAccountStatus(auth *Auth) {
	if auth == nil || auth.Metadata == nil {
		return
	}
	current := PersistedAccountStatus(auth.Metadata)
	switch {
	case auth.Status == StatusBanned || current == StatusBanned:
		SetPersistedAccountStatus(auth, StatusBanned, auth.StatusMessage)
	case auth.Disabled || auth.Status == StatusDisabled || current == StatusDisabled:
		if auth.Disabled || auth.Status == StatusDisabled {
			SetPersistedAccountStatus(auth, StatusDisabled, auth.StatusMessage)
		} else {
			ClearPersistedAccountStatus(auth)
		}
	default:
		ClearPersistedAccountStatus(auth)
	}
}

// ClearPersistedAccountStatus removes any persisted account status markers.
func ClearPersistedAccountStatus(auth *Auth) {
	SetPersistedAccountStatus(auth, StatusActive, "")
}

// AccountStatus returns the user-visible account status independent of the enabled toggle.
func AccountStatus(auth *Auth) Status {
	if auth == nil {
		return StatusUnknown
	}
	if auth.Status == StatusBanned || PersistedAccountStatus(auth.Metadata) == StatusBanned {
		return StatusBanned
	}
	return StatusActive
}

// AccountStatusString returns "banned" for banned auths and "normal" otherwise.
func AccountStatusString(auth *Auth) string {
	if AccountStatus(auth) == StatusBanned {
		return string(StatusBanned)
	}
	return "normal"
}

// MarkAuthBanned mutates an auth entry into a blocked banned account state.
func MarkAuthBanned(auth *Auth, message string, now time.Time) {
	if auth == nil {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "token invalidated"
	}
	if auth.Metadata == nil {
		auth.Metadata = make(map[string]any)
	}
	auth.Status = StatusBanned
	auth.StatusMessage = message
	auth.Unavailable = true
	auth.NextRetryAfter = time.Time{}
	auth.NextRefreshAfter = time.Time{}
	auth.LastError = &Error{
		Code:       "auth_banned",
		Message:    message,
		Retryable:  false,
		HTTPStatus: 401,
	}
	auth.UpdatedAt = now
	SetPersistedAccountStatus(auth, StatusBanned, message)
}

// IsBlocked reports whether the auth is disabled or banned.
func (a *Auth) IsBlocked() bool {
	if a == nil {
		return true
	}
	return a.Disabled || IsBlockedStatus(a.Status) || a.IsBanned()
}

// IsBanned reports whether the auth is banned either in runtime state or persisted metadata.
func (a *Auth) IsBanned() bool {
	if a == nil {
		return false
	}
	if a.Status == StatusBanned {
		return true
	}
	return PersistedAccountStatus(a.Metadata) == StatusBanned
}

// IsBlockedStatus reports whether a status value should be treated as blocked.
func IsBlockedStatus(status Status) bool {
	return status == StatusDisabled || status == StatusBanned
}

// IsAuthenticationTokenInvalidatedError reports whether an error is a Codex token invalidation.
func IsAuthenticationTokenInvalidatedError(err error) bool {
	if err == nil {
		return false
	}
	status := 0
	type statusCoder interface {
		StatusCode() int
	}
	var sc statusCoder
	if errors.As(err, &sc) && sc != nil {
		status = sc.StatusCode()
	}
	return IsAuthenticationTokenInvalidated(status, err.Error())
}

// IsAuthenticationTokenInvalidated reports whether a response indicates an invalidated token.
func IsAuthenticationTokenInvalidated(status int, body string) bool {
	lower := strings.ToLower(strings.TrimSpace(body))
	if lower == "" {
		return false
	}
	if status != 0 && status != 401 {
		return false
	}
	if status == 401 && strings.Contains(lower, "token") && strings.Contains(lower, "invalidated") {
		return true
	}
	if strings.Contains(lower, "401") && strings.Contains(lower, "token") && strings.Contains(lower, "invalidated") {
		return true
	}
	return strings.Contains(lower, "your authentication token has been invalidated")
}

// TokenInvalidatedMessage formats a short persisted message for banned accounts.
func TokenInvalidatedMessage(errOrBody any) string {
	switch v := errOrBody.(type) {
	case nil:
		return "token invalidated"
	case error:
		if s := strings.TrimSpace(v.Error()); s != "" {
			return s
		}
	case string:
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	default:
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" {
			return s
		}
	}
	return "token invalidated"
}

func normalizePersistedAccountStatus(raw any) string {
	switch v := raw.(type) {
	case string:
		return normalizePersistedAccountStatusString(v)
	case []byte:
		return normalizePersistedAccountStatusString(string(v))
	default:
		if raw == nil {
			return ""
		}
		return normalizePersistedAccountStatusString(fmt.Sprint(raw))
	}
}

func normalizePersistedAccountStatusString(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "active", "normal":
		return ""
	case persistedAccountStatusBanned:
		return persistedAccountStatusBanned
	case persistedAccountStatusDisabled:
		return persistedAccountStatusDisabled
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}
