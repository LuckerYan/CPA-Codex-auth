package management

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	codexjwt "github.com/router-for-me/CLIProxyAPI/v6/internal/auth/codex"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/registry"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/executor"
)

type fakeCodexValidationExecutor struct {
	invalid   map[string]bool
	delay     time.Duration
	active    int32
	maxActive int32
}

func (e *fakeCodexValidationExecutor) Identifier() string {
	return "codex"
}

func (e *fakeCodexValidationExecutor) Execute(ctx context.Context, auth *coreauth.Auth, req cliproxyexecutor.Request, opts cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	if auth == nil {
		return cliproxyexecutor.Response{}, &coreauth.Error{Code: "auth_not_found", Message: "missing auth"}
	}
	current := atomic.AddInt32(&e.active, 1)
	for {
		maxSeen := atomic.LoadInt32(&e.maxActive)
		if current <= maxSeen || atomic.CompareAndSwapInt32(&e.maxActive, maxSeen, current) {
			break
		}
	}
	defer atomic.AddInt32(&e.active, -1)

	if e.delay > 0 {
		timer := time.NewTimer(e.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return cliproxyexecutor.Response{}, ctx.Err()
		case <-timer.C:
		}
	}
	if e.invalid[auth.ID] {
		return cliproxyexecutor.Response{}, &coreauth.Error{
			Code:       "auth_unavailable",
			Message:    "unauthorized",
			HTTPStatus: http.StatusUnauthorized,
		}
	}
	return cliproxyexecutor.Response{Payload: []byte(`{"ok":true}`)}, nil
}

func (e *fakeCodexValidationExecutor) ExecuteStream(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (*cliproxyexecutor.StreamResult, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *fakeCodexValidationExecutor) Refresh(_ context.Context, auth *coreauth.Auth) (*coreauth.Auth, error) {
	return auth, nil
}

func (e *fakeCodexValidationExecutor) CountTokens(context.Context, *coreauth.Auth, cliproxyexecutor.Request, cliproxyexecutor.Options) (cliproxyexecutor.Response, error) {
	return cliproxyexecutor.Response{}, nil
}

func (e *fakeCodexValidationExecutor) HttpRequest(ctx context.Context, auth *coreauth.Auth, req *http.Request) (*http.Response, error) {
	if auth == nil {
		return nil, &coreauth.Error{Code: "auth_not_found", Message: "missing auth"}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	current := atomic.AddInt32(&e.active, 1)
	for {
		maxSeen := atomic.LoadInt32(&e.maxActive)
		if current <= maxSeen || atomic.CompareAndSwapInt32(&e.maxActive, maxSeen, current) {
			break
		}
	}
	defer atomic.AddInt32(&e.active, -1)

	if e.delay > 0 {
		timer := time.NewTimer(e.delay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("missing request url")
	}
	if req.Method != http.MethodGet || req.URL.Scheme != "https" || req.URL.Host != "chatgpt.com" || req.URL.Path != "/backend-api/wham/usage" {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     http.StatusText(http.StatusNotFound),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"unexpected quota request"}}`)),
		}, nil
	}
	expectedAccountID := ""
	if auth.Metadata != nil {
		if v, ok := auth.Metadata["account_id"].(string); ok {
			expectedAccountID = strings.TrimSpace(v)
		}
	}
	if expectedAccountID == "" {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"missing expected account id"}}`)),
		}, nil
	}
	if got := strings.TrimSpace(req.Header.Get("Chatgpt-Account-Id")); got != expectedAccountID {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     http.StatusText(http.StatusBadRequest),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"missing or mismatched Chatgpt-Account-Id"}}`)),
		}, nil
	}
	if e.invalid[auth.ID] {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Status:     http.StatusText(http.StatusUnauthorized),
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"401 Your authentication token has been invalidated. Please try signing in again."}}`)),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"plan_type":"free","rate_limit":{},"code_review_rate_limit":{},"additional_rate_limits":[]}`)),
	}, nil
}

func TestSummarizeCodexCardsIncludesRedeemedToday(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, loc)
	redeemedToday := time.Date(2026, 5, 5, 17, 30, 0, 0, time.UTC)
	redeemedYesterday := time.Date(2026, 5, 5, 15, 30, 0, 0, time.UTC)
	cards := []*codexCardRecord{
		{Status: codexCardStatusUnused},
		{Status: codexCardStatusRedeemed, RedeemedAt: &redeemedToday},
		{Status: codexCardStatusRedeemed, RedeemedAt: &redeemedYesterday},
		{Status: codexCardStatusRedeemed},
		{Status: codexCardStatusDisabled},
	}

	summary := summarizeCodexCards(cards, now)

	want := map[string]int{
		"total":          5,
		"unused":         1,
		"redeemed":       3,
		"redeemed_today": 1,
		"disabled":       1,
	}
	for key, expected := range want {
		if summary[key] != expected {
			t.Fatalf("summary[%s] = %d, want %d (summary=%+v)", key, summary[key], expected, summary)
		}
	}
}

func TestValidateCodexAuthCandidatesBansInvalidatedQuotaAuth(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	executor := &fakeCodexValidationExecutor{invalid: map[string]bool{"bad.json": true}}
	manager.RegisterExecutor(executor)

	ctx := context.Background()
	auths := []string{"bad.json", "good-a.json", "good-b.json"}
	for _, id := range auths {
		registerTestCodexAuthRecord(t, manager, id, "")
	}

	h := &Handler{authManager: manager}
	selected, err := h.validateCodexAuthCandidates(ctx, []codexAuthCandidate{
		{ID: "bad.json", FileName: "bad.json"},
		{ID: "good-a.json", FileName: "good-a.json"},
		{ID: "good-b.json", FileName: "good-b.json"},
	}, 2)
	if err != nil {
		t.Fatalf("validate candidates failed: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("expected 2 selected auths, got %d", len(selected))
	}
	for _, candidate := range selected {
		if candidate.ID == "bad.json" {
			t.Fatalf("unauthorized candidate was selected: %+v", selected)
		}
	}
	badAuth, ok := manager.GetByID("bad.json")
	if !ok {
		t.Fatalf("expected bad auth to remain registered")
	}
	if !badAuth.IsBlocked() || badAuth.Status != coreauth.StatusBanned {
		t.Fatalf("expected bad auth to be banned after invalidation, got status=%s blocked=%v message=%q", badAuth.Status, badAuth.IsBlocked(), badAuth.StatusMessage)
	}
	wantInvalidationMessage := "401 Your authentication token has been invalidated. Please try signing in again."
	if badAuth.StatusMessage != wantInvalidationMessage {
		t.Fatalf("expected banned status message %q, got %q", wantInvalidationMessage, badAuth.StatusMessage)
	}
}

func TestCollectCodexAuthCandidatesSkipsBannedQuotaAuth(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	executor := &fakeCodexValidationExecutor{invalid: map[string]bool{"bad.json": true}}
	manager.RegisterExecutor(executor)

	authDir := t.TempDir()
	badPath := writeTestCodexAuthFile(t, authDir, "bad.json", "bad@example.com")
	goodAPath := writeTestCodexAuthFile(t, authDir, "good-a.json", "good-a@example.com")
	goodBPath := writeTestCodexAuthFile(t, authDir, "good-b.json", "good-b@example.com")

	registerTestCodexAuth(t, manager, "bad.json", badPath)
	registerTestCodexAuth(t, manager, "good-a.json", goodAPath)
	registerTestCodexAuth(t, manager, "good-b.json", goodBPath)

	h := &Handler{cfg: &config.Config{AuthDir: authDir}, authManager: manager}
	selected, err := h.validateCodexAuthCandidates(context.Background(), []codexAuthCandidate{
		{ID: "bad.json", FileName: "bad.json"},
		{ID: "good-a.json", FileName: "good-a.json"},
		{ID: "good-b.json", FileName: "good-b.json"},
	}, 2)
	if err != nil {
		t.Fatalf("validate candidates failed: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("expected 2 selected auths, got %d", len(selected))
	}
	for _, candidate := range selected {
		if candidate.ID == "bad.json" {
			t.Fatalf("banned candidate was selected: %+v", selected)
		}
	}

	bannedAuth, ok := manager.GetByID("bad.json")
	if !ok {
		t.Fatalf("expected bad auth to remain registered")
	}
	if !bannedAuth.IsBlocked() || bannedAuth.Status != coreauth.StatusBanned {
		t.Fatalf("expected bad auth to be banned after invalidation, got status=%s blocked=%v message=%q", bannedAuth.Status, bannedAuth.IsBlocked(), bannedAuth.StatusMessage)
	}
	wantInvalidationMessage := "401 Your authentication token has been invalidated. Please try signing in again."
	if bannedAuth.StatusMessage != wantInvalidationMessage {
		t.Fatalf("expected banned status message %q, got %q", wantInvalidationMessage, bannedAuth.StatusMessage)
	}

	candidates, err := h.collectCodexAuthCandidates(context.Background(), map[string]struct{}{})
	if err != nil {
		t.Fatalf("collect candidates failed after ban: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates after skipping banned auth, got %d: %+v", len(candidates), candidates)
	}
	for _, candidate := range candidates {
		if candidate.ID == "bad.json" {
			t.Fatalf("banned candidate should not be collected again: %+v", candidates)
		}
	}
}

func TestValidateCodexAuthCandidatesDoesNotRequireRegistryModelRegistration(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(&fakeCodexValidationExecutor{invalid: map[string]bool{}})

	const authID = "codex-no-registry.json"
	registry.GetGlobalRegistry().UnregisterClient(authID)
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(authID)
	})
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:       authID,
		Provider: "codex",
		Status:   coreauth.StatusActive,
		Metadata: map[string]any{
			"type":         "codex",
			"account_id":   "account-" + authID,
			"access_token": "access-" + authID,
		},
	}); err != nil {
		t.Fatalf("register auth: %v", err)
	}
	if registry.GetGlobalRegistry().ClientSupportsModel(authID, codexValidationModel) {
		t.Fatalf("test setup expected %s to have no registry model registration", authID)
	}

	h := &Handler{authManager: manager}
	selected, err := h.validateCodexAuthCandidates(context.Background(), []codexAuthCandidate{
		{ID: authID, FileName: authID},
	}, 1)
	if err != nil {
		t.Fatalf("validate candidates failed without registry model registration: %v", err)
	}
	if len(selected) != 1 || selected[0].ID != authID {
		t.Fatalf("unexpected selected candidates: %+v", selected)
	}
}

func TestValidateCodexAuthCandidatesRunsConcurrently(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	executor := &fakeCodexValidationExecutor{invalid: map[string]bool{}, delay: 60 * time.Millisecond}
	manager.RegisterExecutor(executor)

	ctx := context.Background()
	candidates := make([]codexAuthCandidate, 0, 4)
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("auth-%d.json", i)
		registerTestCodexAuthRecord(t, manager, id, "")
		candidates = append(candidates, codexAuthCandidate{ID: id, FileName: id})
	}

	h := &Handler{authManager: manager}
	selected, err := h.validateCodexAuthCandidates(ctx, candidates, 4)
	if err != nil {
		t.Fatalf("validate candidates failed: %v", err)
	}
	if len(selected) != 4 {
		t.Fatalf("expected 4 selected auths, got %d", len(selected))
	}
	if maxActive := atomic.LoadInt32(&executor.maxActive); maxActive < 2 {
		t.Fatalf("expected concurrent validation, max active=%d", maxActive)
	}
}

func TestValidateCodexAuthCandidatesSearchesConcurrentlyWhenNeedIsOne(t *testing.T) {
	manager := coreauth.NewManager(nil, nil, nil)
	invalid := make(map[string]bool)
	candidates := make([]codexAuthCandidate, 0, 4)
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("auth-%d.json", i)
		invalid[id] = true
		registerTestCodexAuthRecord(t, manager, id, "")
		candidates = append(candidates, codexAuthCandidate{ID: id, FileName: id})
	}
	executor := &fakeCodexValidationExecutor{invalid: invalid, delay: 60 * time.Millisecond}
	manager.RegisterExecutor(executor)

	h := &Handler{authManager: manager}
	_, err := h.validateCodexAuthCandidates(context.Background(), candidates, 1)
	if err == nil {
		t.Fatalf("expected validation failure when every auth is invalid")
	}
	if maxActive := atomic.LoadInt32(&executor.maxActive); maxActive < 2 {
		t.Fatalf("expected concurrent validation while searching for one valid auth, max active=%d", maxActive)
	}
}

func TestExtractCodexAuthFilesReturnsZipAndRedeemsCards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authDir := t.TempDir()

	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(&fakeCodexValidationExecutor{invalid: map[string]bool{"codex-bad.json": true}})

	validA := writeTestCodexAuthFile(t, authDir, "codex-a.json", "a@example.com")
	validB := writeTestCodexAuthFile(t, authDir, "codex-b.json", "b@example.com")
	bad := writeTestCodexAuthFile(t, authDir, "codex-bad.json", "bad@example.com")

	registerTestCodexAuth(t, manager, "codex-a.json", validA)
	registerTestCodexAuth(t, manager, "codex-b.json", validB)
	registerTestCodexAuth(t, manager, "codex-bad.json", bad)

	store, err := getCodexCardStore(authDir)
	if err != nil {
		t.Fatalf("get card store: %v", err)
	}
	if _, _, _, errImport := store.importCodes([]string{"card-a", "card-b"}); errImport != nil {
		t.Fatalf("import cards: %v", errImport)
	}

	h := &Handler{
		cfg:         &config.Config{AuthDir: authDir},
		authManager: manager,
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v0/management/codex-extract", strings.NewReader(`{"items":["https://email-verification-worker.1330257897.workers.dev/token-code?email=a@example.com&key=CARD-A","https://email-verification-worker.1330257897.workers.dev/token-code?email=b@example.com&key=CARD-B"]}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExtractCodexAuthFiles(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "application/zip") {
		t.Fatalf("expected zip content type, got %q", contentType)
	}

	zipReader, errZip := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if errZip != nil {
		t.Fatalf("read zip: %v", errZip)
	}
	entries := make(map[string]string)
	for _, file := range zipReader.File {
		rc, errOpen := file.Open()
		if errOpen != nil {
			t.Fatalf("open zip entry %s: %v", file.Name, errOpen)
		}
		buf := new(bytes.Buffer)
		if _, errCopy := buf.ReadFrom(rc); errCopy != nil {
			_ = rc.Close()
			t.Fatalf("read zip entry %s: %v", file.Name, errCopy)
		}
		_ = rc.Close()
		entries[file.Name] = buf.String()
	}
	if _, ok := entries["codex-bad.json"]; ok {
		t.Fatalf("unauthorized auth file should not be included: %+v", entries)
	}
	for _, name := range []string{"codex-a.json", "codex-b.json"} {
		if _, ok := entries[name]; !ok {
			t.Fatalf("expected zip entry %s, got %+v", name, entries)
		}
	}
	if !strings.Contains(entries["codex-a.json"], `"email":"a@example.com"`) {
		t.Fatalf("codex-a content did not match original download JSON: %s", entries["codex-a.json"])
	}

	cards, errList := store.list()
	if errList != nil {
		t.Fatalf("list cards: %v", errList)
	}
	redeemed := 0
	for _, card := range cards {
		if card.Status == codexCardStatusRedeemed {
			redeemed++
			if card.RedeemedFile == "" || card.RedeemedAuthID == "" {
				t.Fatalf("redeemed card missing audit fields: %+v", card)
			}
		}
	}
	if redeemed != 2 {
		t.Fatalf("expected 2 redeemed cards, got %d: %+v", redeemed, cards)
	}
}

func TestExtractCodexAuthFilesReturnsSubJSONAndRedeemsCards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authDir := t.TempDir()

	manager := coreauth.NewManager(nil, nil, nil)
	manager.RegisterExecutor(&fakeCodexValidationExecutor{invalid: map[string]bool{}})

	validA := writeTestCodexAuthFile(t, authDir, "codex-a.json", "a@example.com")
	registerTestCodexAuth(t, manager, "codex-a.json", validA)

	store, err := getCodexCardStore(authDir)
	if err != nil {
		t.Fatalf("get card store: %v", err)
	}
	if _, _, _, errImport := store.importCodes([]string{"card-a"}); errImport != nil {
		t.Fatalf("import cards: %v", errImport)
	}

	h := &Handler{
		cfg:         &config.Config{AuthDir: authDir},
		authManager: manager,
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v0/management/codex-extract", strings.NewReader(`{"items":["CARD-A"],"format":"sub"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.ExtractCodexAuthFiles(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if contentType := w.Header().Get("Content-Type"); !strings.Contains(contentType, "application/json") {
		t.Fatalf("expected json content type, got %q", contentType)
	}
	if disposition := w.Header().Get("Content-Disposition"); !strings.Contains(disposition, "sub2api-account-") || !strings.Contains(disposition, ".json") {
		t.Fatalf("expected sub2api json download name, got %q", disposition)
	}

	var payload codexSubExport
	if errJSON := json.Unmarshal(w.Body.Bytes(), &payload); errJSON != nil {
		t.Fatalf("unmarshal sub export: %v\n%s", errJSON, w.Body.String())
	}
	if payload.ExportedAt == "" {
		t.Fatalf("expected exported_at to be set")
	}
	if len(payload.Proxies) != 0 {
		t.Fatalf("expected empty proxies array, got %+v", payload.Proxies)
	}
	if len(payload.Accounts) != 1 {
		t.Fatalf("expected one exported account, got %+v", payload.Accounts)
	}
	account := payload.Accounts[0]
	if account.Name != "a@example.com" || account.Platform != "openai" || account.Type != "oauth" {
		t.Fatalf("unexpected sub account identity: %+v", account)
	}
	if account.Credentials.AccessToken != "access-codex-a.json" || account.Credentials.RefreshToken != "refresh-codex-a.json" {
		t.Fatalf("unexpected sub credentials: %+v", account.Credentials)
	}
	if account.Concurrency != 100 || account.Priority != 1 || account.RateMultiplier != 1 || !account.AutoPauseOnExpired {
		t.Fatalf("unexpected sub account defaults: %+v", account)
	}

	cards, errList := store.list()
	if errList != nil {
		t.Fatalf("list cards: %v", errList)
	}
	if len(cards) != 1 || cards[0].Status != codexCardStatusRedeemed {
		t.Fatalf("expected card to be redeemed after sub export, got %+v", cards)
	}
}

func TestBuildCodexAuthSubJSONMatchesSub2APIShape(t *testing.T) {
	accessToken := makeTestJWT(t, map[string]any{
		"client_id": codexjwt.ClientID,
		"exp":       1778701192,
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "account-sub",
			"chatgpt_plan_type":  "plus",
			"chatgpt_user_id":    "user-sub",
			"user_id":            "user-sub",
		},
		"https://api.openai.com/profile": map[string]any{
			"email":          "sub@example.com",
			"email_verified": true,
		},
	})
	idToken := makeTestJWT(t, map[string]any{
		"aud":            []string{codexjwt.ClientID},
		"email":          "sub@example.com",
		"email_verified": true,
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "account-sub",
			"chatgpt_plan_type":  "plus",
			"chatgpt_user_id":    "user-sub",
			"organizations": []map[string]any{{
				"id":         "org-sub",
				"is_default": true,
				"role":       "owner",
				"title":      "Personal",
			}},
			"user_id": "user-sub",
		},
	})
	body, errMarshal := json.Marshal(map[string]any{
		"id_token":                 idToken,
		"access_token":             accessToken,
		"refresh_token":            "rt-sub",
		"account_id":               "account-sub",
		"last_refresh":             "2026-01-01T00:00:00Z",
		"email":                    "sub@example.com",
		"type":                     "codex",
		"expired":                  "2026-05-14T03:39:52+08:00",
		"websockets":               true,
		"codex_5h_used_percent":    44,
		"codex_7d_used_percent":    75,
		"codex_usage_updated_at":   "2026-05-04T21:03:39+08:00",
		"openai_oauth_passthrough": false,
		"openai_passthrough":       true,
		"privacy_mode":             "training_off",
		"concurrency":              100,
		"priority":                 1,
		"rate_multiplier":          1,
		"auto_pause_on_expired":    true,
		"model_mapping":            map[string]any{},
	})
	if errMarshal != nil {
		t.Fatalf("marshal test auth: %v", errMarshal)
	}

	data, name, err := buildCodexAuthSubJSON([]codexSelectedAuth{{
		AuthID:   "codex-sub@example.com.json",
		FileName: "codex-sub@example.com.json",
		Data:     body,
	}})
	if err != nil {
		t.Fatalf("build sub json: %v", err)
	}
	if !strings.HasPrefix(name, "sub2api-account-") || !strings.HasSuffix(name, ".json") {
		t.Fatalf("unexpected sub json file name: %s", name)
	}

	raw := string(data)
	for _, want := range []string{
		`"exported_at":`,
		`"proxies": []`,
		`"accounts": [`,
		`"name": "sub@example.com"`,
		`"platform": "openai"`,
		`"type": "oauth"`,
		`"credentials": {`,
		`"model_mapping": {}`,
		`"extra": {`,
		`"auto_pause_on_expired": true`,
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("sub json missing %q:\n%s", want, raw)
		}
	}

	var payload codexSubExport
	if errJSON := json.Unmarshal(data, &payload); errJSON != nil {
		t.Fatalf("unmarshal sub export: %v", errJSON)
	}
	if len(payload.Accounts) != 1 {
		t.Fatalf("expected one account, got %+v", payload.Accounts)
	}
	account := payload.Accounts[0]
	if account.Credentials.ChatgptAccountID != "account-sub" {
		t.Fatalf("unexpected chatgpt account id: %+v", account.Credentials)
	}
	if account.Credentials.ChatgptUserID != "user-sub" {
		t.Fatalf("unexpected chatgpt user id: %+v", account.Credentials)
	}
	if account.Credentials.ClientID != codexjwt.ClientID {
		t.Fatalf("unexpected client id: %+v", account.Credentials)
	}
	if account.Credentials.OrganizationID != "org-sub" || account.Credentials.PlanType != "plus" {
		t.Fatalf("unexpected org/plan fields: %+v", account.Credentials)
	}
	if account.Extra.Codex5HWindowMinutes != 300 || account.Extra.Codex7DWindowMinutes != 10080 {
		t.Fatalf("unexpected default codex windows: %+v", account.Extra)
	}
	if account.Extra.Codex5HUsedPercent != 44 || account.Extra.Codex7DUsedPercent != 75 {
		t.Fatalf("unexpected copied codex usage: %+v", account.Extra)
	}
	if !account.Extra.OpenAIOAuthResponsesWebsocketsV2 || account.Extra.OpenAIOAuthResponsesWebsocketsV2Mode != "ctx_pool" {
		t.Fatalf("unexpected websocket defaults: %+v", account.Extra)
	}
}

func TestCodexSubExtraFromUsageMapsWhamUsageResponse(t *testing.T) {
	raw := []byte(`{
  "user_id": "user-YaTVgCePTqVDon9Z47GlgLt4",
  "account_id": "user-YaTVgCePTqVDon9Z47GlgLt4",
  "email": "avorx472as@lucker-yan.asia",
  "plan_type": "plus",
  "rate_limit": {
    "allowed": true,
    "limit_reached": false,
    "primary_window": {
      "used_percent": 0,
      "limit_window_seconds": 18000,
      "reset_after_seconds": 18000,
      "reset_at": 1777923843
    },
    "secondary_window": {
      "used_percent": 76,
      "limit_window_seconds": 604800,
      "reset_after_seconds": 355033,
      "reset_at": 1778260876
    }
  },
  "code_review_rate_limit": null,
  "additional_rate_limits": null,
  "credits": {
    "has_credits": false,
    "unlimited": false,
    "overage_limit_reached": false,
    "balance": "0"
  },
  "spend_control": {
    "reached": false,
    "individual_limit": null
  },
  "rate_limit_reached_type": null,
  "promo": null,
  "referral_beacon": null
}`)
	var usage codexQuotaUsageResponse
	if err := json.Unmarshal(raw, &usage); err != nil {
		t.Fatalf("unmarshal usage response: %v", err)
	}

	extra := codexSubExtraFromUsage(map[string]any{
		"openai_oauth_passthrough": false,
		"openai_passthrough":       true,
		"privacy_mode":             "training_off",
	}, &usage)

	if got, want := extra.Codex5HResetAfterSeconds, 18000; got != want {
		t.Fatalf("codex_5h_reset_after_seconds = %d, want %d", got, want)
	}
	if got, want := extra.Codex5HWindowMinutes, 300; got != want {
		t.Fatalf("codex_5h_window_minutes = %d, want %d", got, want)
	}
	if got, want := extra.Codex5HUsedPercent, 0; got != want {
		t.Fatalf("codex_5h_used_percent = %d, want %d", got, want)
	}
	if got, want := extra.Codex5HResetAt, time.Unix(1777923843, 0).In(time.Local).Format(time.RFC3339); got != want {
		t.Fatalf("codex_5h_reset_at = %s, want %s", got, want)
	}
	if got, want := extra.Codex7DResetAfterSeconds, 355033; got != want {
		t.Fatalf("codex_7d_reset_after_seconds = %d, want %d", got, want)
	}
	if got, want := extra.Codex7DWindowMinutes, 10080; got != want {
		t.Fatalf("codex_7d_window_minutes = %d, want %d", got, want)
	}
	if got, want := extra.Codex7DUsedPercent, 76; got != want {
		t.Fatalf("codex_7d_used_percent = %d, want %d", got, want)
	}
	if got, want := extra.Codex7DResetAt, time.Unix(1778260876, 0).In(time.Local).Format(time.RFC3339); got != want {
		t.Fatalf("codex_7d_reset_at = %s, want %s", got, want)
	}
	if got, want := extra.CodexPrimaryOverSecondaryPercent, 0; got != want {
		t.Fatalf("codex_primary_over_secondary_percent = %d, want %d", got, want)
	}
	if got, want := extra.CodexPrimaryResetAfterSeconds, 18000; got != want {
		t.Fatalf("codex_primary_reset_after_seconds = %d, want %d", got, want)
	}
	if got, want := extra.CodexPrimaryUsedPercent, 0; got != want {
		t.Fatalf("codex_primary_used_percent = %d, want %d", got, want)
	}
	if got, want := extra.CodexSecondaryResetAfterSeconds, 355033; got != want {
		t.Fatalf("codex_secondary_reset_after_seconds = %d, want %d", got, want)
	}
	if got, want := extra.CodexSecondaryUsedPercent, 76; got != want {
		t.Fatalf("codex_secondary_used_percent = %d, want %d", got, want)
	}
	if got, want := extra.CodexSecondaryWindowMinutes, 10080; got != want {
		t.Fatalf("codex_secondary_window_minutes = %d, want %d", got, want)
	}
	if extra.CodexUsageUpdatedAt == "" {
		t.Fatalf("expected codex_usage_updated_at to be populated")
	}
	if extra.OpenAIOAuthPassthrough != false || extra.OpenAIPassthrough != true || extra.PrivacyMode != "training_off" {
		t.Fatalf("unexpected passthrough/privacy flags: %+v", extra)
	}
}

func TestNormalizeCodexCardCodeValidatedExtractsURLKey(t *testing.T) {
	raw := "https://email-verification-worker.1330257897.workers.dev/token-code?email=rzdsqn00pt@lucker-yan.asia&key=et_GHihiHG0SSKIx1q4UCpfAA"
	got, ok := normalizeCodexCardCodeValidated(raw)
	if !ok {
		t.Fatalf("expected URL card code to be valid")
	}
	if want := "et_GHihiHG0SSKIx1q4UCpfAA"; got != want {
		t.Fatalf("unexpected URL key extraction: got %q want %q", got, want)
	}
}

func TestNormalizeCodexCardCodeValidatedExtractsDashDelimitedKeycodeURL(t *testing.T) {
	raw := "0buktk8sl6@thinktank.edu.kg---https://mail.lucker.cc.cd/keycode?email=0buktk8sl6@thinktank.edu.kg&key=et_1QcTaQFX3QFXxTGVzS5ztQ"
	got, ok := normalizeCodexCardCodeValidated(raw)
	if !ok {
		t.Fatalf("expected dash-delimited keycode URL to be valid")
	}
	if want := "et_1QcTaQFX3QFXxTGVzS5ztQ"; got != want {
		t.Fatalf("unexpected dash-delimited keycode extraction: got %q want %q", got, want)
	}
}

func TestNormalizeCodexCardCodeValidatedPreservesWorkerKeyCase(t *testing.T) {
	got, ok := normalizeCodexCardCodeValidated("et_GHihiHG0SSKIx1q4UCpfAA")
	if !ok {
		t.Fatalf("expected worker key to be valid")
	}
	if want := "et_GHihiHG0SSKIx1q4UCpfAA"; got != want {
		t.Fatalf("unexpected worker key normalization: got %q want %q", got, want)
	}
}

func TestNormalizeCodexCardCodeValidatedKeepsLegacyUppercaseBehavior(t *testing.T) {
	got, ok := normalizeCodexCardCodeValidated("cdx-abcdef")
	if !ok {
		t.Fatalf("expected legacy card code to be valid")
	}
	if want := "CDX-ABCDEF"; got != want {
		t.Fatalf("unexpected legacy normalization: got %q want %q", got, want)
	}

	got, ok = normalizeCodexCardCodeValidated("card-a")
	if !ok {
		t.Fatalf("expected legacy external card code to be valid")
	}
	if want := "CARD-A"; got != want {
		t.Fatalf("unexpected external legacy normalization: got %q want %q", got, want)
	}
}

func TestCodexCardStoreImportExtractsURLKey(t *testing.T) {
	authDir := t.TempDir()
	store, err := getCodexCardStore(authDir)
	if err != nil {
		t.Fatalf("get card store: %v", err)
	}
	raw := "0buktk8sl6@thinktank.edu.kg---https://mail.lucker.cc.cd/keycode?email=0buktk8sl6@thinktank.edu.kg&key=et_GHihiHG0SSKIx1q4UCpfAA"
	added, duplicates, invalid, errImport := store.importCodes([]string{raw})
	if errImport != nil {
		t.Fatalf("import cards: %v", errImport)
	}
	if len(duplicates) != 0 || len(invalid) != 0 {
		t.Fatalf("unexpected duplicate/invalid import result: duplicates=%v invalid=%v", duplicates, invalid)
	}
	if len(added) != 1 {
		t.Fatalf("expected one imported card, got %+v", added)
	}
	if want := "et_GHihiHG0SSKIx1q4UCpfAA"; added[0].Code != want {
		t.Fatalf("unexpected imported code: got %q want %q", added[0].Code, want)
	}
}

func TestCollectCodexAuthCandidatesSkipsRedeemedAuthFiles(t *testing.T) {
	authDir := t.TempDir()

	manager := coreauth.NewManager(nil, nil, nil)
	redeemedPath := writeTestCodexAuthFile(t, authDir, "codex-redeemed.json", "redeemed@example.com")
	availablePath := writeTestCodexAuthFile(t, authDir, "codex-available.json", "available@example.com")
	registerTestCodexAuth(t, manager, "codex-redeemed.json", redeemedPath)
	registerTestCodexAuth(t, manager, "codex-available.json", availablePath)

	store, err := getCodexCardStore(authDir)
	if err != nil {
		t.Fatalf("get card store: %v", err)
	}
	if _, _, _, errImport := store.importCodes([]string{"card-used"}); errImport != nil {
		t.Fatalf("import cards: %v", errImport)
	}
	if errRedeem := store.redeem([]string{"card-used"}, []codexSelectedAuth{{
		AuthID:   "codex-redeemed.json",
		FileName: "codex-redeemed.json",
		FilePath: redeemedPath,
		Data:     []byte(`{"type":"codex"}`),
	}}); errRedeem != nil {
		t.Fatalf("redeem setup failed: %v", errRedeem)
	}

	redeemedKeys, errKeys := store.redeemedAuthKeys()
	if errKeys != nil {
		t.Fatalf("redeemed keys: %v", errKeys)
	}
	h := &Handler{
		cfg:         &config.Config{AuthDir: authDir},
		authManager: manager,
	}
	candidates, errCandidates := h.collectCodexAuthCandidates(context.Background(), redeemedKeys)
	if errCandidates != nil {
		t.Fatalf("collect candidates: %v", errCandidates)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected only one unredeemed candidate, got %+v", candidates)
	}
	if candidates[0].FileName != "codex-available.json" {
		t.Fatalf("redeemed auth was not filtered out, got %+v", candidates)
	}
}

func TestCodexAuthClonedContentIsBlockedAndCountedAsRedeemed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authDir := t.TempDir()

	sharedBody := []byte(`{"id_token":"id-shared","access_token":"access-shared","refresh_token":"refresh-shared","account_id":"account-shared","last_refresh":"2026-01-01T00:00:00Z","email":"shared@example.com","type":"codex","expired":"2026-12-31T00:00:00Z"}`)
	uniqueBody := []byte(`{"id_token":"id-unique","access_token":"access-unique","refresh_token":"refresh-unique","account_id":"account-unique","last_refresh":"2026-01-01T00:00:00Z","email":"unique@example.com","type":"codex","expired":"2026-12-31T00:00:00Z"}`)

	pathA := writeTestCodexAuthContent(t, authDir, "codex-a.json", sharedBody)
	pathB := writeTestCodexAuthContent(t, authDir, "codex-b.json", sharedBody)
	pathC := writeTestCodexAuthContent(t, authDir, "codex-c.json", uniqueBody)

	metaShared := mustCodexAuthMetadata(t, sharedBody)
	metaUnique := mustCodexAuthMetadata(t, uniqueBody)

	manager := coreauth.NewManager(nil, nil, nil)
	registerTestCodexAuthWithMetadata(t, manager, "codex-a.json", pathA, metaShared)
	registerTestCodexAuthWithMetadata(t, manager, "codex-b.json", pathB, metaShared)
	registerTestCodexAuthWithMetadata(t, manager, "codex-c.json", pathC, metaUnique)

	store, err := getCodexCardStore(authDir)
	if err != nil {
		t.Fatalf("get card store: %v", err)
	}
	if _, _, _, errImport := store.importCodes([]string{"card-a", "card-b"}); errImport != nil {
		t.Fatalf("import cards: %v", errImport)
	}

	if errRedeem := store.redeem([]string{"card-a"}, []codexSelectedAuth{{
		AuthID:          "codex-a.json",
		FileName:        filepath.Base(pathA),
		FilePath:        pathA,
		Data:            sharedBody,
		ReservationKeys: codexAuthReservationKeys("codex-a.json", filepath.Base(pathA), pathA, metaShared),
	}}); errRedeem != nil {
		t.Fatalf("first redeem failed: %v", errRedeem)
	}

	errRedeemClone := store.redeem([]string{"card-b"}, []codexSelectedAuth{{
		AuthID:          "codex-b.json",
		FileName:        filepath.Base(pathB),
		FilePath:        pathB,
		Data:            sharedBody,
		ReservationKeys: codexAuthReservationKeys("codex-b.json", filepath.Base(pathB), pathB, metaShared),
	}})
	if errRedeemClone == nil {
		t.Fatalf("expected cloned auth redeem to fail")
	}
	if !strings.Contains(strings.ToLower(errRedeemClone.Error()), "already redeemed") {
		t.Fatalf("expected already redeemed error, got %v", errRedeemClone)
	}

	redeemedKeys, errKeys := store.redeemedAuthKeys()
	if errKeys != nil {
		t.Fatalf("redeemed keys: %v", errKeys)
	}
	h := &Handler{
		cfg:         &config.Config{AuthDir: authDir},
		authManager: manager,
	}
	candidates, errCandidates := h.collectCodexAuthCandidates(context.Background(), redeemedKeys)
	if errCandidates != nil {
		t.Fatalf("collect candidates: %v", errCandidates)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected one available candidate after clone filtering, got %+v", candidates)
	}
	if candidates[0].ID != "codex-c.json" {
		t.Fatalf("expected unique auth to remain available, got %+v", candidates)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v0/management/auth-files?is_webui=1", nil)

	h.ListAuthFiles(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from ListAuthFiles, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Files          []map[string]any `json:"files"`
		CodexAuthStats map[string]int   `json:"codex_auth_stats"`
	}
	if errJSON := json.Unmarshal(w.Body.Bytes(), &resp); errJSON != nil {
		t.Fatalf("unmarshal auth file list: %v", errJSON)
	}
	if got := resp.CodexAuthStats["total"]; got != 3 {
		t.Fatalf("expected total 3, got %d", got)
	}
	if got := resp.CodexAuthStats["extracted"]; got != 2 {
		t.Fatalf("expected extracted 2, got %d", got)
	}
	if got := resp.CodexAuthStats["unextracted"]; got != 1 {
		t.Fatalf("expected unextracted 1, got %d", got)
	}

	filesByName := make(map[string]map[string]any, len(resp.Files))
	for _, file := range resp.Files {
		name, _ := file["name"].(string)
		if name != "" {
			filesByName[name] = file
		}
	}
	if fileB, ok := filesByName[filepath.Base(pathB)]; !ok {
		t.Fatalf("expected cloned auth file entry to exist in auth list: %+v", filesByName)
	} else if redeemed, _ := fileB["codex_redeemed"].(bool); !redeemed {
		t.Fatalf("expected cloned auth file to be marked redeemed, got %+v", fileB)
	}
}

func TestCodexCardStoreRedeemRejectsAlreadyRedeemedAuth(t *testing.T) {
	authDir := t.TempDir()
	store, err := getCodexCardStore(authDir)
	if err != nil {
		t.Fatalf("get card store: %v", err)
	}
	if _, _, _, errImport := store.importCodes([]string{"card-a", "card-b"}); errImport != nil {
		t.Fatalf("import cards: %v", errImport)
	}
	file := codexSelectedAuth{
		AuthID:   "codex-a.json",
		FileName: "codex-a.json",
		FilePath: filepath.Join(authDir, "codex-a.json"),
		Data:     []byte(`{"type":"codex"}`),
	}
	if errRedeem := store.redeem([]string{"card-a"}, []codexSelectedAuth{file}); errRedeem != nil {
		t.Fatalf("first redeem failed: %v", errRedeem)
	}
	errRedeemAgain := store.redeem([]string{"card-b"}, []codexSelectedAuth{file})
	if errRedeemAgain == nil {
		t.Fatalf("expected duplicate auth redeem to fail")
	}
	if !strings.Contains(errRedeemAgain.Error(), "already redeemed") {
		t.Fatalf("expected already redeemed error, got %v", errRedeemAgain)
	}
}

func TestCodexCardStoreDeleteReleasesRedeemedAuthReservation(t *testing.T) {
	authDir := t.TempDir()
	store, err := getCodexCardStore(authDir)
	if err != nil {
		t.Fatalf("get card store: %v", err)
	}
	if _, _, _, errImport := store.importCodes([]string{"card-a", "card-b"}); errImport != nil {
		t.Fatalf("import cards: %v", errImport)
	}
	file := codexSelectedAuth{
		AuthID:   "codex-a.json",
		FileName: "codex-a.json",
		FilePath: filepath.Join(authDir, "codex-a.json"),
		Data:     []byte(`{"type":"codex"}`),
	}
	if errRedeem := store.redeem([]string{"card-a"}, []codexSelectedAuth{file}); errRedeem != nil {
		t.Fatalf("redeem failed: %v", errRedeem)
	}
	deleted, notFound, errDelete := store.deleteCodes([]string{"card-a"})
	if errDelete != nil {
		t.Fatalf("delete cards: %v", errDelete)
	}
	if len(deleted) != 1 || deleted[0] != "CARD-A" || len(notFound) != 0 {
		t.Fatalf("unexpected delete result: deleted=%v notFound=%v", deleted, notFound)
	}
	cards, errList := store.list()
	if errList != nil {
		t.Fatalf("list cards: %v", errList)
	}
	if len(cards) != 1 || cards[0].Code != "CARD-B" {
		t.Fatalf("expected only CARD-B to remain, got %+v", cards)
	}
	errRedeemAgain := store.redeem([]string{"card-b"}, []codexSelectedAuth{file})
	if errRedeemAgain != nil {
		t.Fatalf("expected deleting redeemed card to release auth reservation, got %v", errRedeemAgain)
	}
}

func writeTestCodexAuthFile(t *testing.T, dir, name, email string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	body := fmt.Sprintf(`{"id_token":"id-%s","access_token":"access-%s","refresh_token":"refresh-%s","account_id":"account-%s","last_refresh":"2026-01-01T00:00:00Z","email":"%s","type":"codex","expired":"2026-12-31T00:00:00Z"}`+"\n", name, name, name, name, email)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write auth file %s: %v", name, err)
	}
	return path
}

func registerTestCodexAuth(t *testing.T, manager *coreauth.Manager, id, path string) {
	t.Helper()
	registerTestCodexAuthRecord(t, manager, id, path)
}

func registerTestCodexAuthRecord(t *testing.T, manager *coreauth.Manager, id, path string) {
	t.Helper()
	attrs := map[string]string{}
	fileName := id
	if path != "" {
		attrs["path"] = path
		fileName = filepath.Base(path)
	}
	accountID := "account-" + fileName
	_, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:         id,
		Provider:   "codex",
		FileName:   fileName,
		Status:     coreauth.StatusActive,
		Attributes: attrs,
		Metadata:   map[string]any{"type": "codex", "account_id": accountID, "access_token": "access-" + fileName},
	})
	if err != nil {
		t.Fatalf("register auth %s: %v", id, err)
	}
	registry.GetGlobalRegistry().RegisterClient(id, "codex", []*registry.ModelInfo{{ID: codexValidationModel}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(id)
	})
}

func writeTestCodexAuthContent(t *testing.T, dir, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write auth file %s: %v", name, err)
	}
	return path
}

func mustCodexAuthMetadata(t *testing.T, body []byte) map[string]any {
	t.Helper()
	metadata := make(map[string]any)
	if err := json.Unmarshal(body, &metadata); err != nil {
		t.Fatalf("unmarshal auth metadata: %v", err)
	}
	return metadata
}

func registerTestCodexAuthWithMetadata(t *testing.T, manager *coreauth.Manager, id, path string, metadata map[string]any) {
	t.Helper()
	attrs := map[string]string{"path": path}
	_, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:         id,
		Provider:   "codex",
		FileName:   filepath.Base(path),
		Status:     coreauth.StatusActive,
		Attributes: attrs,
		Metadata:   cloneTestMetadata(metadata),
	})
	if err != nil {
		t.Fatalf("register auth %s: %v", id, err)
	}
}

func cloneTestMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func makeTestJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	header := map[string]any{"alg": "none", "typ": "JWT"}
	headerData, errHeader := json.Marshal(header)
	if errHeader != nil {
		t.Fatalf("marshal jwt header: %v", errHeader)
	}
	claimsData, errClaims := json.Marshal(claims)
	if errClaims != nil {
		t.Fatalf("marshal jwt claims: %v", errClaims)
	}
	return base64.RawURLEncoding.EncodeToString(headerData) + "." + base64.RawURLEncoding.EncodeToString(claimsData) + ".signature"
}
