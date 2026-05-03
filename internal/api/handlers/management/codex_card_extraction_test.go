package management

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
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

func (e *fakeCodexValidationExecutor) HttpRequest(context.Context, *coreauth.Auth, *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

func TestValidateCodexAuthCandidatesSkipsUnauthorized(t *testing.T) {
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
	raw := "https://email-verification-worker.1330257897.workers.dev/token-code?email=rzdsqn00pt@lucker-yan.asia&key=et_GHihiHG0SSKIx1q4UCpfAA"
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
	_, err := manager.Register(context.Background(), &coreauth.Auth{
		ID:         id,
		Provider:   "codex",
		FileName:   fileName,
		Status:     coreauth.StatusActive,
		Attributes: attrs,
		Metadata:   map[string]any{"type": "codex"},
	})
	if err != nil {
		t.Fatalf("register auth %s: %v", id, err)
	}
	registry.GetGlobalRegistry().RegisterClient(id, "codex", []*registry.ModelInfo{{ID: codexValidationModel}})
	t.Cleanup(func() {
		registry.GetGlobalRegistry().UnregisterClient(id)
	})
}
