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
	assertCodexExtractionPageContains(t, "CPA 格式")
	assertCodexExtractionPageContains(t, "SUB 格式")
	assertCodexExtractionPageNotContains(t, "（当前）")
	assertCodexExtractionPageNotContains(t, "多个 Codex 认证 JSON 打包成 ZIP；每个账号保持独立文件。")
	assertCodexExtractionPageNotContains(t, "生成单个 sub2api-account JSON；多个账号合并在 accounts 数组内，不压缩。")
	assertCodexExtractionPageNotContains(t, "sub2api-account JSON")
	assertCodexExtractionPageContains(t, "function getSelectedFormat()")
	assertCodexExtractionPageContains(t, "input[name=\"extractFormat\"]")
	assertCodexExtractionPageContains(t, "JSON.stringify({ items: codes, format: format })")
	assertCodexExtractionPageContains(t, ".center .panel { transform: translateY(clamp(-30px, -2.2vw, -18px)); }")
	assertCodexExtractionPageContains(t, ".center .panel { transform: none; }")
	assertCodexExtractionPageContains(t, "id=\"progressShell\"")
	assertCodexExtractionPageContains(t, "role=\"progressbar\"")
	assertCodexExtractionPageContains(t, ".progress-shell { position: fixed;")
	assertCodexExtractionPageContains(t, "top: 50%; bottom: auto;")
	assertCodexExtractionPageContains(t, "transform: translate(-50%, -50%);")
	assertCodexExtractionPageContains(t, ".progress-shell[hidden] { display: none; }")
	assertCodexExtractionPageContains(t, "function startProgress")
	assertCodexExtractionPageContains(t, "function completeProgress")
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
