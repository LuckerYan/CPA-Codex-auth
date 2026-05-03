package management

import (
	"strings"
	"testing"
)

func TestCodexExtractionPageSupportsBatchCodes(t *testing.T) {
	assertCodexExtractionPageContains(t, "<textarea id=\"cardCode\"")
	assertCodexExtractionPageNotContains(t, "一行一个，支持 token-code 链接")
	assertCodexExtractionPageContains(t, "token-code?email")
	assertCodexExtractionPageContains(t, "function getCardCodes()")
	assertCodexExtractionPageContains(t, "function extractCardCodeInput")
	assertCodexExtractionPageContains(t, "searchParams.get('key')")
	assertCodexExtractionPageContains(t, ".map(extractCardCodeInput)")
	assertCodexExtractionPageContains(t, "JSON.stringify({ items: codes })")
	assertCodexExtractionPageContains(t, "验活中（")
	assertCodexExtractionPageContains(t, "event.ctrlKey || event.metaKey")
}

func assertCodexExtractionPageContains(t *testing.T, want string) {
	t.Helper()
	if !strings.Contains(codexExtractionPageHTML, want) {
		t.Fatalf("extraction page does not contain %q", want)
	}
}

func assertCodexExtractionPageNotContains(t *testing.T, want string) {
	t.Helper()
	if strings.Contains(codexExtractionPageHTML, want) {
		t.Fatalf("extraction page still contains %q", want)
	}
}
