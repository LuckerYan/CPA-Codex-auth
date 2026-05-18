package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPatchAuthFilesAgainstRealManagementHTML guards against silent breakage
// when upstream re-minifies the React bundle and renames identifiers (e.g.
// Yx→Zx, Vv→Uv, s→$e). It loads the bundled static/management.html and
// verifies that both fallback patches successfully inject the Codex-specific
// helpers, so the card-code search and Plus/Free filters keep working.
func TestPatchAuthFilesAgainstRealManagementHTML(t *testing.T) {
	wd, _ := os.Getwd()
	htmlPath := filepath.Join(wd, "..", "..", "static", "management.html")
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Skipf("management.html not available: %v", err)
	}
	patched := patchQuotaManagementPanel(data)
	if !strings.Contains(string(patched), "cardBatchSearchMarker=`__codex_card_batch__=`") {
		t.Errorf("search fallback failed to apply against real management.html")
	}
	if !strings.Contains(string(patched), "codexPlusFilterMatch") {
		t.Errorf("filter fallback failed to apply against real management.html")
	}
	if !strings.Contains(string(patched), "cardBatchActiveForFilters") {
		t.Errorf("cardBatchActiveForFilters helper not injected")
	}
}
