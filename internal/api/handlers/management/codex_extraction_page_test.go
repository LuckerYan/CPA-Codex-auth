package management

import (
	"strings"
	"testing"
)

func TestCodexExtractionPageSupportsBatchCodes(t *testing.T) {
	assertCodexExtractionPageContains(t, "<textarea id=\"cardCode\"")
	assertCodexExtractionPageContains(t, "卡密（一行一个，支持批量）")
	assertCodexExtractionPageContains(t, "function getCardCodes()")
	assertCodexExtractionPageContains(t, "split(/\\r?\\n/)")
	assertCodexExtractionPageContains(t, "JSON.stringify({ items: codes })")
	assertCodexExtractionPageContains(t, "正在批量验活")
	assertCodexExtractionPageContains(t, "event.ctrlKey || event.metaKey")
}

func assertCodexExtractionPageContains(t *testing.T, want string) {
	t.Helper()
	if !strings.Contains(codexExtractionPageHTML, want) {
		t.Fatalf("extraction page does not contain %q", want)
	}
}
