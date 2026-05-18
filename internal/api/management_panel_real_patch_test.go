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
	if !strings.Contains(string(patched), "=1e6,") {
		t.Errorf("quota too-many threshold fallback failed to bump the value")
	}
	if !strings.Contains(string(patched), "/*__cpaQuotaConcurrentMap*/") {
		t.Errorf("quota load concurrency fallback failed to rewrite Promise.all")
	}
	if !strings.Contains(string(patched), "[q,z]=(0,y.useState)(``)") {
		t.Errorf("quota page-size input fallback failed to inject [q,z] state")
	}
	if !strings.Contains(string(patched), ".pageSizeSelect,style:{width:160}") {
		t.Errorf("quota page-size input element not injected")
	}
}

// TestQuotaLoadConcurrencyPatchHasBalancedParens guards against syntax errors
// in the injected worker-pool snippet. JS engines fail hard on a single
// mismatched paren, so we verify it locally by counting (/) within the patch
// payload (kept in sync with the literal replacement string).
func TestQuotaLoadConcurrencyPatchHasBalancedParens(t *testing.T) {
	wd, _ := os.Getwd()
	htmlPath := filepath.Join(wd, "..", "..", "static", "management.html")
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Skipf("management.html not available: %v", err)
	}
	patched := string(patchQuotaManagementPanel(data))
	marker := "/*__cpaQuotaConcurrentMap*/"
	idx := strings.Index(patched, marker)
	if idx < 0 {
		t.Fatalf("concurrency patch not applied")
	}
	// Walk backwards from the marker to the start of the rewritten block
	// ("let i=await(async()=>{") and ensure paren counts balance.
	startMarker := "let i=await(async()=>{"
	start := strings.LastIndex(patched[:idx], startMarker)
	if start < 0 {
		t.Fatalf("could not locate start of concurrency patch")
	}
	snippet := patched[start : idx+len(marker)]
	if open, close := strings.Count(snippet, "("), strings.Count(snippet, ")"); open != close {
		t.Errorf("paren count mismatch in concurrency patch: ( = %d, ) = %d\nsnippet=%q", open, close, snippet)
	}
	if open, close := strings.Count(snippet, "{"), strings.Count(snippet, "}"); open != close {
		t.Errorf("brace count mismatch in concurrency patch: { = %d, } = %d\nsnippet=%q", open, close, snippet)
	}
}
