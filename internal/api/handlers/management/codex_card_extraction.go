package management

import (
	"archive/zip"
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	codexjwt "github.com/router-for-me/CLIProxyAPI/v6/internal/auth/codex"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/util"
	coreauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
	log "github.com/sirupsen/logrus"
)

const (
	codexCardStoreFileName        = ".codex-card-store.db"
	codexCardStatusUnused         = "unused"
	codexCardStatusRedeemed       = "redeemed"
	codexCardStatusDisabled       = "disabled"
	codexValidationModel          = "gpt-5.4-mini"
	codexValidationConcurrencyCap = 16
	codexQuotaUsageURL            = "https://chatgpt.com/backend-api/wham/usage"
	codexQuotaUserAgent           = "codex_cli_rs/0.76.0 (Debian 13.0.0; x86_64) WindowsTerminal"
)

var codexCardStores sync.Map

type codexCardStoreFile struct {
	UpdatedAt        time.Time                   `json:"updated_at"`
	Cards            map[string]*codexCardRecord `json:"cards"`
	RedeemedAuthKeys []string                    `json:"redeemed_auth_keys,omitempty"`
}

type codexCardRecord struct {
	Code             string     `json:"code"`
	Source           string     `json:"source"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	RedeemedAt       *time.Time `json:"redeemed_at,omitempty"`
	RedeemedFile     string     `json:"redeemed_file,omitempty"`
	RedeemedAuthID   string     `json:"redeemed_auth_id,omitempty"`
	RedeemedAuthKeys []string   `json:"redeemed_auth_keys,omitempty"`
	Note             string     `json:"note,omitempty"`
}

type codexAuthCandidate struct {
	ID              string
	FilePath        string
	FileName        string
	ReservationKeys []string
}

type codexSelectedAuth struct {
	CardCode        string
	AuthID          string
	FilePath        string
	FileName        string
	Data            []byte
	ReservationKeys []string
}

type codexCardGenerateRequest struct {
	Count int `json:"count"`
}

type codexCardImportRequest struct {
	Codes string   `json:"codes"`
	Cards string   `json:"cards"`
	Items []string `json:"items"`
}

type codexAuthExtractRequest struct {
	Codes string   `json:"codes"`
	Cards string   `json:"cards"`
	Items []string `json:"items"`
}

type codexCardBatchRequest struct {
	Codes string   `json:"codes"`
	Cards string   `json:"cards"`
	Items []string `json:"items"`
	All   bool     `json:"all"`
}

type codexCardGenerateResponse struct {
	Status    string             `json:"status"`
	Generated int                `json:"generated"`
	Cards     []*codexCardRecord `json:"cards"`
	Codes     []string           `json:"codes"`
}

type codexCardImportResponse struct {
	Status     string             `json:"status"`
	Imported   int                `json:"imported"`
	Duplicates []string           `json:"duplicates,omitempty"`
	Invalid    []string           `json:"invalid,omitempty"`
	Cards      []*codexCardRecord `json:"cards,omitempty"`
	Codes      []string           `json:"codes,omitempty"`
}

type codexCardListResponse struct {
	Status  string             `json:"status"`
	Total   int                `json:"total"`
	Summary map[string]int     `json:"summary"`
	Cards   []*codexCardRecord `json:"cards"`
}

type codexCardDeleteResponse struct {
	Status   string   `json:"status"`
	Deleted  int      `json:"deleted"`
	Codes    []string `json:"codes,omitempty"`
	NotFound []string `json:"not_found,omitempty"`
}

type codexCardStore struct {
	mu                 sync.Mutex
	path               string
	loaded             bool
	updatedAt          time.Time
	cards              map[string]*codexCardRecord
	redeemedAuthKeySet map[string]struct{}
}

func getCodexCardStore(authDir string) (*codexCardStore, error) {
	path, err := codexCardStorePath(authDir)
	if err != nil {
		return nil, err
	}
	value, ok := codexCardStores.Load(path)
	if ok {
		if store, okCast := value.(*codexCardStore); okCast && store != nil {
			return store, nil
		}
	}
	store := &codexCardStore{
		path:               path,
		cards:              make(map[string]*codexCardRecord),
		redeemedAuthKeySet: make(map[string]struct{}),
	}
	actual, _ := codexCardStores.LoadOrStore(path, store)
	if loaded, okCast := actual.(*codexCardStore); okCast && loaded != nil {
		return loaded, nil
	}
	return store, nil
}

func codexCardStorePath(authDir string) (string, error) {
	resolved, err := util.ResolveAuthDir(strings.TrimSpace(authDir))
	if err != nil {
		return "", fmt.Errorf("resolve auth dir: %w", err)
	}
	if strings.TrimSpace(resolved) == "" {
		return "", fmt.Errorf("auth dir is not configured")
	}
	return filepath.Join(resolved, codexCardStoreFileName), nil
}

func (s *codexCardStore) ensureLoadedLocked() error {
	if s == nil {
		return fmt.Errorf("card store is nil")
	}
	if s.loaded {
		if s.cards == nil {
			s.cards = make(map[string]*codexCardRecord)
		}
		if s.redeemedAuthKeySet == nil {
			s.redeemedAuthKeySet = make(map[string]struct{})
		}
		return nil
	}
	s.cards = make(map[string]*codexCardRecord)
	s.redeemedAuthKeySet = make(map[string]struct{})
	data, errRead := os.ReadFile(s.path)
	if errRead != nil {
		if os.IsNotExist(errRead) {
			s.loaded = true
			return nil
		}
		return fmt.Errorf("read card store: %w", errRead)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		s.loaded = true
		return nil
	}
	var payload codexCardStoreFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode card store: %w", err)
	}
	if payload.Cards != nil {
		s.cards = make(map[string]*codexCardRecord, len(payload.Cards))
		for code, record := range payload.Cards {
			if strings.TrimSpace(code) == "" || record == nil {
				continue
			}
			normalized := normalizeCodexCardCode(code)
			if normalized == "" {
				continue
			}
			copyRecord := cloneCodexCardRecord(record)
			copyRecord.Code = normalized
			if copyRecord.Status == "" {
				copyRecord.Status = codexCardStatusUnused
			}
			s.cards[normalized] = copyRecord
		}
	}
	for _, key := range payload.RedeemedAuthKeys {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if normalizedKey != "" {
			s.redeemedAuthKeySet[normalizedKey] = struct{}{}
		}
	}
	s.updatedAt = payload.UpdatedAt
	s.loaded = true
	return nil
}

func (s *codexCardStore) saveLocked() error {
	if s == nil {
		return fmt.Errorf("card store is nil")
	}
	if strings.TrimSpace(s.path) == "" {
		return fmt.Errorf("card store path is empty")
	}
	if s.cards == nil {
		s.cards = make(map[string]*codexCardRecord)
	}
	payload := codexCardStoreFile{
		UpdatedAt:        s.updatedAt,
		Cards:            s.cards,
		RedeemedAuthKeys: s.persistedRedeemedAuthKeysLocked(),
	}
	data, errMarshal := json.MarshalIndent(payload, "", "  ")
	if errMarshal != nil {
		return fmt.Errorf("encode card store: %w", errMarshal)
	}
	if errMkdir := os.MkdirAll(filepath.Dir(s.path), 0o700); errMkdir != nil {
		return fmt.Errorf("prepare card store dir: %w", errMkdir)
	}
	if errWrite := os.WriteFile(s.path, data, 0o600); errWrite != nil {
		return fmt.Errorf("write card store: %w", errWrite)
	}
	return nil
}

func (s *codexCardStore) persistedRedeemedAuthKeysLocked() []string {
	keys := s.redeemedAuthKeysLocked()
	out := make([]string, 0, len(keys))
	for key := range keys {
		if strings.TrimSpace(key) != "" {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func (s *codexCardStore) list() ([]*codexCardRecord, error) {
	if s == nil {
		return nil, fmt.Errorf("card store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return nil, err
	}
	out := make([]*codexCardRecord, 0, len(s.cards))
	for _, record := range s.cards {
		out = append(out, cloneCodexCardRecord(record))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return strings.ToLower(out[i].Code) < strings.ToLower(out[j].Code)
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *codexCardStore) generate(count int) ([]*codexCardRecord, error) {
	if s == nil {
		return nil, fmt.Errorf("card store is nil")
	}
	if count <= 0 {
		return nil, fmt.Errorf("count must be greater than zero")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]*codexCardRecord, 0, count)
	for i := 0; i < count; i++ {
		code, errCode := generateCodexCardCode()
		if errCode != nil {
			return nil, errCode
		}
		for {
			if _, exists := s.cards[code]; !exists {
				break
			}
			code, errCode = generateCodexCardCode()
			if errCode != nil {
				return nil, errCode
			}
		}
		record := &codexCardRecord{
			Code:      code,
			Source:    "generated",
			Status:    codexCardStatusUnused,
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.cards[code] = record
		out = append(out, cloneCodexCardRecord(record))
	}
	s.updatedAt = now
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *codexCardStore) importCodes(codes []string) ([]*codexCardRecord, []string, []string, error) {
	if s == nil {
		return nil, nil, nil, fmt.Errorf("card store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return nil, nil, nil, err
	}
	now := time.Now().UTC()
	added := make([]*codexCardRecord, 0, len(codes))
	duplicates := make([]string, 0)
	invalid := make([]string, 0)
	seenInput := make(map[string]struct{}, len(codes))
	for _, raw := range codes {
		code, ok := normalizeCodexCardCodeValidated(raw)
		if !ok {
			invalid = append(invalid, strings.TrimSpace(raw))
			continue
		}
		if _, seen := seenInput[code]; seen {
			duplicates = append(duplicates, code)
			continue
		}
		seenInput[code] = struct{}{}
		if _, exists := s.cards[code]; exists {
			duplicates = append(duplicates, code)
			continue
		}
		record := &codexCardRecord{
			Code:      code,
			Source:    "imported",
			Status:    codexCardStatusUnused,
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.cards[code] = record
		added = append(added, cloneCodexCardRecord(record))
	}
	if len(added) > 0 {
		s.updatedAt = now
		if err := s.saveLocked(); err != nil {
			return nil, nil, nil, err
		}
	}
	return added, duplicates, invalid, nil
}

func (s *codexCardStore) deleteCodes(codes []string) ([]string, []string, error) {
	if s == nil {
		return nil, nil, fmt.Errorf("card store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	deleted := make([]string, 0, len(codes))
	notFound := make([]string, 0)
	for _, raw := range codes {
		code, ok := normalizeCodexCardCodeValidated(raw)
		if !ok {
			continue
		}
		record, exists := s.cards[code]
		if !exists || record == nil {
			notFound = append(notFound, code)
			continue
		}
		if strings.EqualFold(strings.TrimSpace(record.Status), codexCardStatusRedeemed) {
			removeCodexAuthKeys(s.redeemedAuthKeySet, codexAuthRecordKeys(record))
		}
		delete(s.cards, code)
		deleted = append(deleted, code)
	}
	if len(deleted) > 0 {
		s.updatedAt = now
		if err := s.saveLocked(); err != nil {
			return nil, nil, err
		}
	}
	return deleted, notFound, nil
}

func (s *codexCardStore) ensureAvailable(codes []string) error {
	if s == nil {
		return fmt.Errorf("card store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}
	if len(codes) == 0 {
		return fmt.Errorf("no card codes supplied")
	}
	for _, raw := range codes {
		code, ok := normalizeCodexCardCodeValidated(raw)
		if !ok {
			return fmt.Errorf("invalid card code: %q", raw)
		}
		record, exists := s.cards[code]
		if !exists {
			return fmt.Errorf("card not found: %s", code)
		}
		if record == nil {
			return fmt.Errorf("card not found: %s", code)
		}
		if !strings.EqualFold(strings.TrimSpace(record.Status), codexCardStatusUnused) {
			return fmt.Errorf("card already used: %s", code)
		}
	}
	return nil
}

func (s *codexCardStore) redeemedAuthKeys() (map[string]struct{}, error) {
	if s == nil {
		return nil, fmt.Errorf("card store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return nil, err
	}
	return s.redeemedAuthKeysLocked(), nil
}

func (s *codexCardStore) redeemedAuthKeysLocked() map[string]struct{} {
	keys := make(map[string]struct{})
	if s == nil {
		return keys
	}
	for key := range s.redeemedAuthKeySet {
		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		if normalizedKey != "" {
			keys[normalizedKey] = struct{}{}
		}
	}
	for _, record := range s.cards {
		if record == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(record.Status), codexCardStatusRedeemed) {
			continue
		}
		addCodexAuthKeys(keys, codexAuthRecordKeys(record))
	}
	return keys
}

func (s *codexCardStore) redeem(codes []string, files []codexSelectedAuth) error {
	if s == nil {
		return fmt.Errorf("card store is nil")
	}
	if len(codes) != len(files) {
		return fmt.Errorf("card count and file count mismatch")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, raw := range codes {
		code, ok := normalizeCodexCardCodeValidated(raw)
		if !ok {
			return fmt.Errorf("invalid card code: %q", raw)
		}
		record, exists := s.cards[code]
		if !exists || record == nil {
			return fmt.Errorf("card not found: %s", code)
		}
		if !strings.EqualFold(strings.TrimSpace(record.Status), codexCardStatusUnused) {
			return fmt.Errorf("card already used: %s", code)
		}
	}
	redeemedKeys := s.redeemedAuthKeysLocked()
	selectedKeys := make(map[string]struct{})
	reservationKeysByIndex := make([][]string, len(files))
	for i, file := range files {
		keys := file.ReservationKeys
		if len(keys) == 0 {
			keys = codexAuthReservationKeysFromData(file.Data, file.AuthID, file.FileName, file.FilePath)
		}
		keys = normalizeCodexAuthKeys(keys...)
		reservationKeysByIndex[i] = append([]string(nil), keys...)
		for _, key := range keys {
			if _, used := redeemedKeys[key]; used {
				return fmt.Errorf("codex auth file already redeemed: %s", file.FileName)
			}
			if _, selected := selectedKeys[key]; selected {
				return fmt.Errorf("duplicate codex auth file selected: %s", file.FileName)
			}
		}
		for _, key := range keys {
			selectedKeys[key] = struct{}{}
		}
	}
	for i, raw := range codes {
		code, _ := normalizeCodexCardCodeValidated(raw)
		keys := reservationKeysByIndex[i]
		record := s.cards[code]
		record.Status = codexCardStatusRedeemed
		record.UpdatedAt = now
		redeemedAt := now
		record.RedeemedAt = &redeemedAt
		record.RedeemedFile = files[i].FileName
		record.RedeemedAuthID = files[i].AuthID
		record.RedeemedAuthKeys = append([]string(nil), keys...)
		if s.redeemedAuthKeySet == nil {
			s.redeemedAuthKeySet = make(map[string]struct{})
		}
		addCodexAuthKeys(s.redeemedAuthKeySet, keys)
	}
	s.updatedAt = now
	return s.saveLocked()
}

func cloneCodexCardRecord(record *codexCardRecord) *codexCardRecord {
	if record == nil {
		return nil
	}
	clone := *record
	return &clone
}

func normalizeCodexCardCodeValidated(raw string) (string, bool) {
	trimmed, extractedFromKeyParam := extractCodexCardCodeInput(raw)
	if trimmed == "" {
		return "", false
	}
	if strings.ContainsAny(trimmed, "\r\n\t ") {
		return "", false
	}
	if shouldPreserveCodexCardCodeCase(trimmed, extractedFromKeyParam) {
		return trimmed, true
	}
	return strings.ToUpper(trimmed), true
}

func normalizeCodexCardCode(raw string) string {
	code, ok := normalizeCodexCardCodeValidated(raw)
	if !ok {
		return ""
	}
	return code
}

func extractCodexCardCodeInput(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	for _, candidate := range codexCardCodeInputCandidates(trimmed) {
		if parsed, err := url.Parse(candidate); err == nil && parsed != nil {
			if key := strings.TrimSpace(parsed.Query().Get("key")); key != "" {
				return key, true
			}
		}
		if key := extractCodexCardKeyParamFallback(candidate); key != "" {
			return key, true
		}
	}
	return trimmed, false
}

func codexCardCodeInputCandidates(trimmed string) []string {
	candidates := []string{trimmed}
	markerIndex := strings.Index(trimmed, "---")
	if markerIndex < 0 {
		return candidates
	}
	suffix := strings.TrimSpace(trimmed[markerIndex+3:])
	if suffix == "" || suffix == trimmed {
		return candidates
	}
	return append([]string{suffix}, candidates...)
}

func extractCodexCardKeyParamFallback(raw string) string {
	lower := strings.ToLower(raw)
	markers := []string{"?key=", "&key=", "#key=", "key="}
	for _, marker := range markers {
		idx := strings.Index(lower, marker)
		if marker == "key=" && idx != 0 {
			continue
		}
		if idx < 0 {
			continue
		}
		start := idx + len(marker)
		end := start
		for end < len(raw) {
			switch raw[end] {
			case '&', '#', ' ', '\t', '\r', '\n':
				value := raw[start:end]
				return decodeCodexCardKeyParamValue(value)
			default:
				end++
			}
		}
		return decodeCodexCardKeyParamValue(raw[start:end])
	}
	return ""
}

func decodeCodexCardKeyParamValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	decoded, err := url.QueryUnescape(trimmed)
	if err != nil {
		return trimmed
	}
	return strings.TrimSpace(decoded)
}

func shouldPreserveCodexCardCodeCase(code string, extractedFromKeyParam bool) bool {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "et_") || strings.HasPrefix(trimmed, "et-") {
		return true
	}
	if extractedFromKeyParam && !strings.HasPrefix(strings.ToUpper(trimmed), "CDX-") {
		return true
	}
	return false
}

func codexAuthSelectionKeys(authID, fileName, filePath string) []string {
	rawValues := []string{authID, fileName, filePath}
	seen := make(map[string]struct{})
	keys := make([]string, 0, len(rawValues)*2)
	for _, raw := range rawValues {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		values := []string{trimmed}
		base := filepath.Base(filepath.Clean(trimmed))
		if base != "" && base != "." && base != string(filepath.Separator) {
			values = append(values, base)
		}
		for _, value := range values {
			key := strings.ToLower(strings.TrimSpace(value))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	return keys
}

func codexAuthAlreadyRedeemed(redeemed map[string]struct{}, candidate codexAuthCandidate) bool {
	if len(redeemed) == 0 {
		return false
	}
	for _, key := range codexAuthCandidateKeys(candidate) {
		if _, ok := redeemed[key]; ok {
			return true
		}
	}
	return false
}

func codexAuthCandidateKeys(candidate codexAuthCandidate) []string {
	keys := codexAuthSelectionKeys(candidate.ID, candidate.FileName, candidate.FilePath)
	keys = append(keys, candidate.ReservationKeys...)
	return normalizeCodexAuthKeys(keys...)
}

func codexAuthReservationKeys(authID, fileName, filePath string, metadata map[string]any) []string {
	keys := codexAuthSelectionKeys(authID, fileName, filePath)
	if contentKey := codexAuthContentReservationKey(metadata); contentKey != "" {
		keys = append(keys, contentKey)
	}
	return normalizeCodexAuthKeys(keys...)
}

func codexAuthReservationKeysFromData(data []byte, authID, fileName, filePath string) []string {
	if len(data) == 0 {
		return codexAuthSelectionKeys(authID, fileName, filePath)
	}
	var metadata map[string]any
	if err := json.Unmarshal(data, &metadata); err != nil {
		return codexAuthSelectionKeys(authID, fileName, filePath)
	}
	return codexAuthReservationKeys(authID, fileName, filePath, metadata)
}

func codexAuthContentReservationKey(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	fingerprint := codexAuthContentFingerprint(metadata)
	if fingerprint == "" {
		return ""
	}
	return "content:sha256:" + fingerprint
}

func codexAuthContentFingerprint(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	email := strings.ToLower(strings.TrimSpace(codexAuthMetadataValue(metadata, "email")))
	accountID := strings.ToLower(strings.TrimSpace(codexAuthMetadataValue(metadata, "account_id")))
	refreshToken := strings.TrimSpace(codexAuthMetadataValue(metadata, "refresh_token"))
	idToken := strings.TrimSpace(codexAuthMetadataValue(metadata, "id_token"))
	if email == "" && accountID == "" && refreshToken == "" && idToken == "" {
		return ""
	}
	components := []string{
		strings.ToLower(strings.TrimSpace(codexAuthMetadataValue(metadata, "type"))),
		email,
		accountID,
		refreshToken,
		idToken,
	}
	var builder strings.Builder
	for _, component := range components {
		builder.WriteString(component)
		builder.WriteByte(0x1f)
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(sum[:])
}

func codexAuthMetadataValue(metadata map[string]any, key string) string {
	if v := stringValue(metadata, key); v != "" {
		return v
	}
	if len(metadata) == 0 {
		return ""
	}
	tokenRaw, ok := metadata["token"]
	if !ok || tokenRaw == nil {
		return ""
	}
	switch typed := tokenRaw.(type) {
	case map[string]any:
		if v, ok := typed[key].(string); ok {
			return strings.TrimSpace(v)
		}
	case map[string]string:
		if v := strings.TrimSpace(typed[key]); v != "" {
			return v
		}
	}
	return ""
}

func codexAuthRecordKeys(record *codexCardRecord) []string {
	if record == nil {
		return nil
	}
	keys := append([]string(nil), record.RedeemedAuthKeys...)
	keys = append(keys, codexAuthSelectionKeys(record.RedeemedAuthID, record.RedeemedFile, "")...)
	return normalizeCodexAuthKeys(keys...)
}

func normalizeCodexAuthKeys(keys ...string) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, raw := range keys {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func addCodexAuthKeys(keys map[string]struct{}, values []string) {
	if keys == nil {
		return
	}
	for _, key := range normalizeCodexAuthKeys(values...) {
		keys[key] = struct{}{}
	}
}

func removeCodexAuthKeys(keys map[string]struct{}, values []string) {
	if keys == nil {
		return
	}
	for _, target := range normalizeCodexAuthKeys(values...) {
		delete(keys, target)
	}
}

func splitCodexCardInput(req codexCardImportRequest) []string {
	return splitCodexCardStrings(req.Codes, req.Cards, req.Items)
}

func splitCodexCardExtractInput(req codexAuthExtractRequest) []string {
	return splitCodexCardStrings(req.Codes, req.Cards, req.Items)
}

func splitCodexCardBatchInput(req codexCardBatchRequest) []string {
	return splitCodexCardStrings(req.Codes, req.Cards, req.Items)
}

func splitCodexCardStrings(rawStrings ...interface{}) []string {
	out := make([]string, 0)
	for _, raw := range rawStrings {
		switch v := raw.(type) {
		case string:
			out = append(out, splitCodexCardText(v)...)
		case []string:
			for _, item := range v {
				out = append(out, splitCodexCardText(item)...)
			}
		}
	}
	return out
}

func splitCodexCardText(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	normalized := strings.ReplaceAll(trimmed, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	out := make([]string, 0)
	for _, item := range strings.Split(normalized, "\n") {
		line := strings.TrimSpace(item)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func uniqueCodexCardCodes(codes []string) ([]string, []string, []string) {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	duplicates := make([]string, 0)
	invalid := make([]string, 0)
	for _, raw := range codes {
		code, ok := normalizeCodexCardCodeValidated(raw)
		if !ok {
			if strings.TrimSpace(raw) != "" {
				invalid = append(invalid, strings.TrimSpace(raw))
			}
			continue
		}
		if _, ok := seen[code]; ok {
			duplicates = append(duplicates, code)
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, duplicates, invalid
}

func (h *Handler) GenerateCodexCards(c *gin.Context) {
	if h == nil || h.cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management config unavailable"})
		return
	}
	var req codexCardGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if req.Count <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "count must be greater than zero"})
		return
	}
	store, err := getCodexCardStore(h.cfg.AuthDir)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	cards, err := store.generate(req.Count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	codes := make([]string, 0, len(cards))
	for _, card := range cards {
		codes = append(codes, card.Code)
	}
	c.JSON(http.StatusOK, codexCardGenerateResponse{
		Status:    "ok",
		Generated: len(cards),
		Cards:     cards,
		Codes:     codes,
	})
}

func (h *Handler) ListCodexCards(c *gin.Context) {
	if h == nil || h.cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management config unavailable"})
		return
	}
	store, err := getCodexCardStore(h.cfg.AuthDir)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	cards, err := store.list()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	summary := map[string]int{
		"total":    len(cards),
		"unused":   0,
		"redeemed": 0,
		"disabled": 0,
	}
	for _, card := range cards {
		if card == nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(card.Status)) {
		case codexCardStatusRedeemed:
			summary["redeemed"]++
		case codexCardStatusDisabled:
			summary["disabled"]++
		default:
			summary["unused"]++
		}
	}
	c.JSON(http.StatusOK, codexCardListResponse{
		Status:  "ok",
		Total:   len(cards),
		Summary: summary,
		Cards:   cards,
	})
}

func (h *Handler) ImportCodexCards(c *gin.Context) {
	if h == nil || h.cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management config unavailable"})
		return
	}
	var req codexCardImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	codes := splitCodexCardInput(req)
	if len(codes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no card codes supplied"})
		return
	}
	store, err := getCodexCardStore(h.cfg.AuthDir)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	added, duplicates, invalid, err := store.importCodes(codes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	importedCodes := make([]string, 0, len(added))
	for _, card := range added {
		importedCodes = append(importedCodes, card.Code)
	}
	c.JSON(http.StatusOK, codexCardImportResponse{
		Status:     "ok",
		Imported:   len(added),
		Duplicates: duplicates,
		Invalid:    invalid,
		Cards:      added,
		Codes:      importedCodes,
	})
}

func (h *Handler) DeleteCodexCards(c *gin.Context) {
	if h == nil || h.cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management config unavailable"})
		return
	}
	var req codexCardBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	store, err := getCodexCardStore(h.cfg.AuthDir)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	codes := splitCodexCardBatchInput(req)
	if req.All {
		cards, errList := store.list()
		if errList != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": errList.Error()})
			return
		}
		codes = make([]string, 0, len(cards))
		for _, card := range cards {
			if card != nil {
				codes = append(codes, card.Code)
			}
		}
	}
	codes, _, invalid := uniqueCodexCardCodes(codes)
	if len(invalid) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card codes", "invalid": invalid})
		return
	}
	if len(codes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no card codes supplied"})
		return
	}
	deleted, notFound, errDelete := store.deleteCodes(codes)
	if errDelete != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errDelete.Error()})
		return
	}
	c.JSON(http.StatusOK, codexCardDeleteResponse{
		Status:   "ok",
		Deleted:  len(deleted),
		Codes:    deleted,
		NotFound: notFound,
	})
}

func (h *Handler) ExportCodexCards(c *gin.Context) {
	if h == nil || h.cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management config unavailable"})
		return
	}
	var req codexCardBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	store, err := getCodexCardStore(h.cfg.AuthDir)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	cards, errList := store.list()
	if errList != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errList.Error()})
		return
	}
	selected := make(map[string]struct{})
	if !req.All {
		codes := splitCodexCardBatchInput(req)
		codes, _, invalid := uniqueCodexCardCodes(codes)
		if len(invalid) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card codes", "invalid": invalid})
			return
		}
		if len(codes) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no card codes supplied"})
			return
		}
		for _, code := range codes {
			selected[code] = struct{}{}
		}
	}
	exported := make([]string, 0, len(cards))
	found := make(map[string]struct{}, len(selected))
	for _, card := range cards {
		if card == nil {
			continue
		}
		code := normalizeCodexCardCode(card.Code)
		if code == "" {
			continue
		}
		if !req.All {
			if _, ok := selected[code]; !ok {
				continue
			}
			found[code] = struct{}{}
		}
		exported = append(exported, code)
	}
	if !req.All {
		notFound := make([]string, 0)
		for code := range selected {
			if _, ok := found[code]; !ok {
				notFound = append(notFound, code)
			}
		}
		if len(notFound) > 0 {
			sort.Strings(notFound)
			c.JSON(http.StatusNotFound, gin.H{"error": "some card codes were not found", "not_found": notFound})
			return
		}
	}
	if len(exported) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no card codes to export"})
		return
	}
	fileName := fmt.Sprintf("codex-cards-%s.txt", time.Now().UTC().Format("20060102-150405"))
	body := strings.Join(exported, "\n") + "\n"
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.Header("Cache-Control", "no-store")
	c.String(http.StatusOK, body)
}

func (h *Handler) ExtractCodexAuthFiles(c *gin.Context) {
	if h == nil || h.cfg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "management config unavailable"})
		return
	}
	if h.authManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "core auth manager unavailable"})
		return
	}
	var req codexAuthExtractRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	cardCodes := splitCodexCardExtractInput(req)
	if len(cardCodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no card codes supplied"})
		return
	}
	cardCodes, duplicates, invalid := uniqueCodexCardCodes(cardCodes)
	if len(invalid) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid card codes", "invalid": invalid})
		return
	}
	if len(duplicates) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "duplicate card codes are not allowed", "duplicates": duplicates})
		return
	}
	store, err := getCodexCardStore(h.cfg.AuthDir)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	if err := store.ensureAvailable(cardCodes); err != nil {
		status := http.StatusBadRequest
		switch {
		case strings.Contains(strings.ToLower(err.Error()), "not found"):
			status = http.StatusNotFound
		case strings.Contains(strings.ToLower(err.Error()), "already used"):
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	redeemedAuths, errRedeemedAuths := store.redeemedAuthKeys()
	if errRedeemedAuths != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errRedeemedAuths.Error()})
		return
	}
	candidates, errCandidates := h.collectCodexAuthCandidates(c.Request.Context(), redeemedAuths)
	if errCandidates != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errCandidates.Error()})
		return
	}
	selected, errSelect := h.validateCodexAuthCandidates(c.Request.Context(), candidates, len(cardCodes))
	if errSelect != nil {
		c.JSON(http.StatusConflict, gin.H{"error": errSelect.Error()})
		return
	}
	files, errLoad := h.loadCodexAuthFiles(selected)
	if errLoad != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errLoad.Error()})
		return
	}
	zipBytes, zipName, errZip := buildCodexAuthZip(files)
	if errZip != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errZip.Error()})
		return
	}
	if errRedeem := store.redeem(cardCodes, files); errRedeem != nil {
		status := http.StatusConflict
		if strings.Contains(strings.ToLower(errRedeem.Error()), "not found") {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": errRedeem.Error()})
		return
	}
	if errWrite := writeCodexAuthZip(c, zipName, zipBytes); errWrite != nil {
		log.WithError(errWrite).Error("failed to write codex auth zip")
		return
	}
}

func (h *Handler) collectCodexAuthCandidates(ctx context.Context, redeemedAuths map[string]struct{}) ([]codexAuthCandidate, error) {
	if h == nil || h.authManager == nil {
		return nil, fmt.Errorf("core auth manager unavailable")
	}
	auths := h.authManager.List()
	if len(auths) == 0 {
		return nil, fmt.Errorf("no auth files available")
	}
	authDir := ""
	if h.cfg != nil {
		authDir = strings.TrimSpace(h.cfg.AuthDir)
	}
	resolvedAuthDir, err := util.ResolveAuthDir(authDir)
	if err != nil {
		return nil, fmt.Errorf("resolve auth dir: %w", err)
	}
	seen := make(map[string]struct{}, len(auths))
	candidates := make([]codexAuthCandidate, 0, len(auths))
	availableCodexFiles := 0
	for _, auth := range auths {
		if auth == nil {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
			continue
		}
		if auth.IsBlocked() {
			continue
		}
		path := resolveCodexAuthPath(auth, resolvedAuthDir)
		if path == "" {
			continue
		}
		if _, errStat := os.Stat(path); errStat != nil {
			continue
		}
		availableCodexFiles++
		cleaned := filepath.Clean(path)
		if _, dup := seen[cleaned]; dup {
			continue
		}
		candidate := codexAuthCandidate{
			ID:              auth.ID,
			FilePath:        cleaned,
			FileName:        filepath.Base(cleaned),
			ReservationKeys: codexAuthReservationKeys(auth.ID, filepath.Base(cleaned), cleaned, auth.Metadata),
		}
		if codexAuthAlreadyRedeemed(redeemedAuths, candidate) {
			continue
		}
		seen[cleaned] = struct{}{}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		if availableCodexFiles > 0 && len(redeemedAuths) > 0 {
			return nil, fmt.Errorf("no unredeemed codex auth files available")
		}
		return nil, fmt.Errorf("no codex auth files available")
	}
	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	return candidates, nil
}

func resolveCodexAuthPath(auth *coreauth.Auth, authDir string) string {
	if auth == nil {
		return ""
	}
	path := strings.TrimSpace(authAttribute(auth, "path"))
	if path != "" {
		return path
	}
	fileName := strings.TrimSpace(auth.FileName)
	if fileName == "" {
		fileName = strings.TrimSpace(auth.ID)
	}
	if fileName == "" {
		return ""
	}
	if filepath.IsAbs(fileName) {
		return fileName
	}
	if authDir == "" {
		return fileName
	}
	return filepath.Join(authDir, fileName)
}

func (h *Handler) validateCodexAuthCandidates(ctx context.Context, candidates []codexAuthCandidate, need int) ([]codexAuthCandidate, error) {
	if need <= 0 {
		return nil, fmt.Errorf("requested card count must be greater than zero")
	}
	if len(candidates) < need {
		return nil, fmt.Errorf("not enough codex auth files available: need %d, have %d", need, len(candidates))
	}
	if h == nil || h.authManager == nil {
		return nil, fmt.Errorf("core auth manager unavailable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(coreauth.WithSkipPersist(ctx))
	defer cancel()

	jobs := make(chan codexAuthCandidate)
	workerCount := len(candidates)
	if workerCount > codexValidationConcurrencyCap {
		workerCount = codexValidationConcurrencyCap
	}
	if workerCount < 1 {
		workerCount = 1
	}

	var (
		mu           sync.Mutex
		selected     = make([]codexAuthCandidate, 0, need)
		firstErr     error
		checked      int
		failed       int
		unauthorized int
		workerWG     sync.WaitGroup
		dispatchDone sync.WaitGroup
	)

	validateOne := func(candidate codexAuthCandidate) bool {
		if candidate.ID == "" {
			return false
		}
		errExec := h.validateCodexQuotaCandidate(ctx, candidate.ID)
		if errExec == nil {
			mu.Lock()
			checked++
			mu.Unlock()
			return true
		}
		status := validationErrorStatus(errExec)
		if coreauth.IsAuthenticationTokenInvalidatedError(errExec) {
			log.Debugf("codex auth %s failed quota validation with invalidated token, banning auth and trying another candidate", candidate.ID)
			h.authManager.MarkBanned(context.Background(), candidate.ID, codexAuthInvalidationMessage(errExec))
		} else if status == http.StatusUnauthorized {
			log.Debugf("codex auth %s failed quota validation with 401, trying another candidate", candidate.ID)
		} else {
			log.Debugf("codex auth %s failed quota validation: %v", candidate.ID, errExec)
		}
		mu.Lock()
		checked++
		failed++
		if status == http.StatusUnauthorized {
			unauthorized++
		}
		if firstErr == nil {
			firstErr = errExec
		}
		mu.Unlock()
		return false
	}

	for i := 0; i < workerCount; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case candidate, ok := <-jobs:
					if !ok {
						return
					}
					if validateOne(candidate) {
						mu.Lock()
						if len(selected) < need {
							selected = append(selected, candidate)
							if len(selected) >= need {
								cancel()
							}
						}
						mu.Unlock()
					}
				}
			}
		}()
	}

	dispatchDone.Add(1)
	go func() {
		defer dispatchDone.Done()
		defer close(jobs)
		for _, candidate := range candidates {
			select {
			case <-ctx.Done():
				return
			case jobs <- candidate:
			}
		}
	}()

	dispatchDone.Wait()
	workerWG.Wait()

	if len(selected) < need {
		details := fmt.Sprintf("need %d, valid %d, checked %d of %d", need, len(selected), checked, len(candidates))
		if failed > 0 {
			details += fmt.Sprintf(", failed %d", failed)
		}
		if unauthorized > 0 {
			details += fmt.Sprintf(", 401 %d", unauthorized)
		}
		if firstErr != nil {
			return nil, fmt.Errorf("not enough valid codex auth files (%s): %w", details, firstErr)
		}
		return nil, fmt.Errorf("not enough valid codex auth files (%s)", details)
	}

	sort.Slice(selected, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(selected[i].FileName))
		right := strings.ToLower(strings.TrimSpace(selected[j].FileName))
		if left == right {
			return selected[i].FilePath < selected[j].FilePath
		}
		return left < right
	})

	return selected, nil
}

func (h *Handler) validateCodexQuotaCandidate(ctx context.Context, authID string) error {
	if h == nil || h.authManager == nil {
		return fmt.Errorf("core auth manager unavailable")
	}
	authID = strings.TrimSpace(authID)
	if authID == "" {
		return fmt.Errorf("auth id is empty")
	}
	auth, okAuth := h.authManager.GetByID(authID)
	if !okAuth || auth == nil {
		return &coreauth.Error{Code: "auth_not_found", Message: "auth not found", HTTPStatus: http.StatusNotFound}
	}
	if !strings.EqualFold(strings.TrimSpace(auth.Provider), "codex") {
		return &coreauth.Error{Code: "provider_not_found", Message: "auth provider is not codex", HTTPStatus: http.StatusBadRequest}
	}
	accountID := resolveCodexQuotaAccountID(auth)
	if accountID == "" {
		return &coreauth.Error{Code: "invalid_request", Message: "Codex credential missing ChatGPT account ID", HTTPStatus: http.StatusBadRequest}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, codexQuotaUsageURL, nil)
	if errReq != nil {
		return fmt.Errorf("build codex quota request: %w", errReq)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", codexQuotaUserAgent)
	req.Header.Set("Chatgpt-Account-Id", accountID)

	resp, errExec := h.authManager.HttpRequest(ctx, auth, req)
	if errExec != nil {
		return errExec
	}
	if resp == nil {
		return fmt.Errorf("codex quota response is empty")
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			log.WithError(errClose).Debug("failed to close codex quota response body")
		}
	}()

	body, errRead := io.ReadAll(resp.Body)
	if errRead != nil {
		return fmt.Errorf("read codex quota response: %w", errRead)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := codexQuotaResponseMessage(body)
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return &coreauth.Error{Code: "auth_unavailable", Message: message, HTTPStatus: resp.StatusCode}
	}
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("parse codex quota response: %w", err)
	}
	if parsed == nil {
		return fmt.Errorf("codex quota response is empty")
	}
	return nil
}

func resolveCodexQuotaAccountID(auth *coreauth.Auth) string {
	if auth == nil {
		return ""
	}
	if v := strings.TrimSpace(codexAuthMetadataValue(auth.Metadata, "account_id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(authAttribute(auth, "account_id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(codexAuthMetadataValue(auth.Metadata, "chatgpt_account_id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(authAttribute(auth, "chatgpt_account_id")); v != "" {
		return v
	}
	for _, token := range []string{
		strings.TrimSpace(codexAuthMetadataValue(auth.Metadata, "id_token")),
		strings.TrimSpace(authAttribute(auth, "id_token")),
	} {
		if token == "" {
			continue
		}
		claims, err := codexjwt.ParseJWTToken(token)
		if err != nil || claims == nil {
			continue
		}
		if v := strings.TrimSpace(claims.CodexAuthInfo.ChatgptAccountID); v != "" {
			return v
		}
	}
	return ""
}

func codexQuotaResponseMessage(body []byte) string {
	message := strings.TrimSpace(string(body))
	if message == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return message
	}
	if extracted := codexErrorMessageFromPayload(payload); extracted != "" {
		return extracted
	}
	return message
}

func codexErrorMessageFromPayload(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	if msg := extractStringField(payload, "message"); msg != "" {
		return msg
	}
	if value, ok := payload["error"]; ok {
		switch typed := value.(type) {
		case map[string]any:
			if msg := extractStringField(typed, "message"); msg != "" {
				return msg
			}
			if msg := extractStringField(typed, "error"); msg != "" {
				return msg
			}
		case string:
			if msg := strings.TrimSpace(typed); msg != "" {
				return msg
			}
		}
	}
	if msg := extractStringField(payload, "error_description"); msg != "" {
		return msg
	}
	return ""
}

func extractStringField(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	msg, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(msg)
}

func validationErrorStatus(err error) int {
	if err == nil {
		return 0
	}
	var statusCoder interface{ StatusCode() int }
	if errors.As(err, &statusCoder) {
		return statusCoder.StatusCode()
	}
	var authErr *coreauth.Error
	if errors.As(err, &authErr) && authErr != nil {
		return authErr.StatusCode()
	}
	return 0
}

func codexAuthInvalidationMessage(err error) string {
	if err == nil {
		return ""
	}
	var authErr *coreauth.Error
	if errors.As(err, &authErr) && authErr != nil {
		if message := strings.TrimSpace(authErr.Message); message != "" {
			return message
		}
	}
	return coreauth.TokenInvalidatedMessage(err)
}

func (h *Handler) loadCodexAuthFiles(candidates []codexAuthCandidate) ([]codexSelectedAuth, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no codex auth candidates selected")
	}
	out := make([]codexSelectedAuth, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.FilePath) == "" {
			return nil, fmt.Errorf("auth file path is empty for %s", candidate.ID)
		}
		data, errRead := os.ReadFile(candidate.FilePath)
		if errRead != nil {
			return nil, fmt.Errorf("read auth file %s: %w", candidate.FilePath, errRead)
		}
		out = append(out, codexSelectedAuth{
			AuthID:          candidate.ID,
			FilePath:        candidate.FilePath,
			FileName:        candidate.FileName,
			Data:            data,
			ReservationKeys: append([]string(nil), candidate.ReservationKeys...),
		})
	}
	return out, nil
}

func buildCodexAuthZip(files []codexSelectedAuth) ([]byte, string, error) {
	if len(files) == 0 {
		return nil, "", fmt.Errorf("no codex auth files selected")
	}
	zipName := fmt.Sprintf("codex-auth-files-%s.zip", time.Now().UTC().Format("20060102-150405"))
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, file := range files {
		name := safeCodexZipEntryName(file.FileName)
		entry, errCreate := writer.Create(name)
		if errCreate != nil {
			_ = writer.Close()
			return nil, "", fmt.Errorf("create zip entry %s: %w", name, errCreate)
		}
		if _, errWrite := io.Copy(entry, bytes.NewReader(file.Data)); errWrite != nil {
			_ = writer.Close()
			return nil, "", fmt.Errorf("write zip entry %s: %w", name, errWrite)
		}
	}
	if errClose := writer.Close(); errClose != nil {
		return nil, "", fmt.Errorf("close zip writer: %w", errClose)
	}
	return append([]byte(nil), buffer.Bytes()...), zipName, nil
}

func writeCodexAuthZip(c *gin.Context, zipName string, zipBytes []byte) error {
	if c == nil {
		return fmt.Errorf("gin context is nil")
	}
	if len(zipBytes) == 0 {
		return fmt.Errorf("no codex auth zip data available")
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", zipName))
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)
	if _, errWrite := c.Writer.Write(zipBytes); errWrite != nil {
		return fmt.Errorf("write zip response: %w", errWrite)
	}
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func safeCodexZipEntryName(name string) string {
	cleaned := filepath.Base(strings.TrimSpace(name))
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) {
		return fmt.Sprintf("codex-auth-%d.json", time.Now().UnixNano())
	}
	if !strings.HasSuffix(strings.ToLower(cleaned), ".json") {
		cleaned += ".json"
	}
	return cleaned
}

func generateCodexCardCode() (string, error) {
	raw := make([]byte, 16)
	if _, err := crand.Read(raw); err != nil {
		return "", fmt.Errorf("generate card code: %w", err)
	}
	return "CDX-" + strings.ToUpper(hex.EncodeToString(raw)), nil
}
