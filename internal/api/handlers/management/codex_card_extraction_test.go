package management

import (
	"archive/zip"
	"bytes"
	"context"
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
