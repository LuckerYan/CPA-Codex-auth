package management

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

// persistCodexPlanType writes the resolved plan_type into the auth's metadata and
// persists the change to disk through the auth manager. It is a no-op when the
// value is empty or already matches what is stored.
func (h *Handler) persistCodexPlanType(ctx context.Context, auth *coreauth.Auth, planType string) {
	if h == nil || h.authManager == nil || auth == nil {
		return
	}
	planType = strings.ToLower(strings.TrimSpace(planType))
	if planType == "" {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		return
	}

	current, _ := h.authManager.GetByID(auth.ID)
	if current == nil {
		current = auth
	}
	if existing := strings.ToLower(strings.TrimSpace(codexAuthMetadataValue(current.Metadata, "plan_type"))); existing == planType {
		return
	}
	if current.Metadata == nil {
		current.Metadata = map[string]any{}
	}
	current.Metadata["plan_type"] = planType

	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := h.authManager.Update(ctx, current); err != nil {
		log.WithError(err).WithField("auth_id", current.ID).Warn("failed to persist codex plan_type")
	}
}

// fetchAndPersistCodexPlanType calls the upstream wham/usage endpoint for the
// given codex auth, parses the plan_type field from the response, and persists
// it through persistCodexPlanType. Designed to be invoked from a goroutine; all
// failures are logged at debug level and swallowed.
func (h *Handler) fetchAndPersistCodexPlanType(ctx context.Context, authID string) {
	if h == nil || h.authManager == nil {
		return
	}
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return
	}
	auth, ok := h.authManager.GetByID(authID)
	if !ok || auth == nil {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	accountID := resolveCodexQuotaAccountID(auth)
	if accountID == "" {
		log.WithField("auth_id", authID).Debug("skip codex plan_type sync: missing account id")
		return
	}

	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, codexQuotaUsageURL, nil)
	if errReq != nil {
		log.WithError(errReq).Debug("build codex plan_type sync request failed")
		return
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", codexQuotaUserAgent)
	req.Header.Set("Chatgpt-Account-Id", accountID)

	resp, errExec := h.authManager.HttpRequest(ctx, auth, req)
	if errExec != nil {
		log.WithError(errExec).WithField("auth_id", authID).Debug("codex plan_type sync upstream call failed")
		return
	}
	if resp == nil {
		return
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			log.WithError(errClose).Debug("close codex plan_type sync response body failed")
		}
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return
	}

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		log.WithError(errRead).Debug("read codex plan_type sync response failed")
		return
	}
	planType := extractCodexPlanTypeFromUsageBody(body)
	if planType == "" {
		return
	}
	h.persistCodexPlanType(ctx, auth, planType)
}

// extractCodexPlanTypeFromUsageBody parses the wham/usage response body and
// returns the plan_type field (lower-cased, trimmed). Empty string when the
// field is absent or the body cannot be parsed.
func extractCodexPlanTypeFromUsageBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var usage codexQuotaUsageResponse
	if err := json.Unmarshal(body, &usage); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(usage.PlanType))
}

// isCodexUsageEndpoint reports whether the proxied URL targets the upstream
// wham/usage quota endpoint. Tolerates trailing slashes and query strings.
func isCodexUsageEndpoint(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	if idx := strings.IndexAny(rawURL, "?#"); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	rawURL = strings.TrimRight(rawURL, "/")
	return strings.HasSuffix(rawURL, "/backend-api/wham/usage")
}
