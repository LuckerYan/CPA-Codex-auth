package management

import (
	"strings"
	"testing"
)

func TestCodexExtractionPageSupportsBatchCodes(t *testing.T) {
	assertCodexExtractionPageContains(t, "<textarea id=\"cardCode\"")
	assertCodexExtractionPageContains(t, "输入卡密，<span class=\"hl\">一键提取</span>")
	assertCodexExtractionPageNotContains(t, "输入卡密或链接")
	assertCodexExtractionPageNotContains(t, "一行一个，支持 token-code 链接")
	assertCodexExtractionPageNotContains(t, "token-code?email")
	assertCodexExtractionPageContains(t, "邮箱---keycode 链接")
	assertCodexExtractionPageContains(t, "mail.lucker.cc.cd/keycode?email")
	assertCodexExtractionPageContains(t, "function getCardCodes()")
	assertCodexExtractionPageContains(t, "function extractCardCodeInput")
	assertCodexExtractionPageContains(t, "function cardCodeInputCandidates")
	assertCodexExtractionPageContains(t, "trimmed.indexOf('---')")
	assertCodexExtractionPageContains(t, "searchParams.get('key')")
	assertCodexExtractionPageContains(t, ".map(extractCardCodeInput)")
	assertCodexExtractionPageContains(t, "JSON.stringify({ items: codes })")
	assertCodexExtractionPageContains(t, "验活中（")
	assertCodexExtractionPageContains(t, "event.ctrlKey || event.metaKey")
	assertCodexExtractionPageContains(t, "请先输入卡密或提取链接")
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
