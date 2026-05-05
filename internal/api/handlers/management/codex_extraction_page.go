package management

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ServeCodexExtractionPage(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(codexExtractionPageHTML))
}

const codexExtractionPageHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CODEX EXTRACT · Codex 认证文件提取</title>
  <link rel="icon" type="image/svg+xml" href="data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 32 32'><defs><linearGradient id='g' x1='0%25' y1='0%25' x2='100%25' y2='100%25'><stop offset='0%25' stop-color='%23fff34d'/><stop offset='42%25' stop-color='%23b9f06e'/><stop offset='100%25' stop-color='%238ed7ef'/></linearGradient></defs><rect width='32' height='32' rx='7' fill='url(%23g)'/><g fill='none' stroke='%2335312d' stroke-width='2.2' stroke-linecap='round' stroke-linejoin='round'><path d='M16 6 8 10.5l8 11.5 8-11.5L16 6Z'/><path d='M16 9.5v9'/><path d='M11.5 11.7h9'/><path d='m12 15.5 4 2.7 4-2.7'/></g></svg>">
  <style>
    :root {
      color-scheme: dark;
      --bg-root: #0f0e0d;
      --bg-panel: #181715;
      --bg-card: #1f1d1a;
      --border: rgba(255,255,255,.105);
      --border-strong: rgba(255,255,255,.16);
      --text-primary: #f5f2ee;
      --text-secondary: #b9b2aa;
      --text-tertiary: #8b8680;
      --primary: #8b8680;
      --accent-a: #fff34d;
      --accent-b: #b9f06e;
      --accent-c: #8ed7ef;
      --success: #10b981;
      --error: #ef9a8b;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      color: var(--text-primary);
      background:
        radial-gradient(circle at 16% 0%, rgba(139,134,128,.20), transparent 34%),
        radial-gradient(circle at 88% 14%, rgba(142,215,239,.08), transparent 32%),
        radial-gradient(circle at 50% 110%, rgba(185,240,110,.06), transparent 50%),
        linear-gradient(180deg, #0d0c0b 0%, var(--bg-root) 42%, #11100f 100%);
      overflow-x: hidden;
    }
    body::before {
      content: "";
      position: fixed;
      inset: 0;
      pointer-events: none;
      background: linear-gradient(90deg, rgba(255,255,255,.025) 1px, transparent 1px), linear-gradient(rgba(255,255,255,.018) 1px, transparent 1px);
      background-size: 48px 48px;
      mask-image: radial-gradient(circle at 50% 30%, black, transparent 70%);
    }
    .shell { min-height: 100vh; display: grid; grid-template-rows: auto 1fr auto; position: relative; z-index: 1; }
    .topbar { display: flex; align-items: center; justify-content: space-between; padding: 26px clamp(20px, 5vw, 72px); gap: 16px; }
    .brand { display: inline-flex; align-items: center; gap: 12px; color: var(--text-primary); font-size: 22px; font-weight: 900; letter-spacing: .04em; }
    .brand .name { background: linear-gradient(135deg, #f5f2ee 0%, #d8d4cf 100%); -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent; }
    .brand .sep { color: rgba(255,255,255,.18); margin: 0 2px; font-weight: 600; }
    .brand .sub { color: var(--text-tertiary); font-size: 12px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; }
    .logo { width: 40px; height: 40px; border-radius: 10px; background: linear-gradient(135deg, var(--accent-a) 0%, var(--accent-b) 42%, var(--accent-c) 100%); box-shadow: 0 14px 40px rgba(141,215,239,.18), inset 0 1px 0 rgba(255,255,255,.5); display: grid; place-items: center; color: #2a2722; }
    .logo svg { width: 24px; height: 24px; }
    .pill { border: 1px solid var(--border); background: rgba(31,29,26,.70); border-radius: 999px; padding: 9px 14px 9px 12px; color: var(--text-secondary); font-size: 13px; display: inline-flex; align-items: center; gap: 8px; backdrop-filter: blur(6px); }
    .dot { width: 9px; height: 9px; background: var(--success); border-radius: 50%; box-shadow: 0 0 0 5px rgba(16,185,129,.12); animation: pulse 2.4s ease-in-out infinite; }
    @keyframes pulse { 0%,100% { box-shadow: 0 0 0 5px rgba(16,185,129,.12); } 50% { box-shadow: 0 0 0 8px rgba(16,185,129,.04); } }
    .center { display: grid; place-items: center; padding: 8px clamp(18px, 5vw, 72px) 48px; }
    .panel { width: min(100%, 960px); border: 1px solid var(--border); background: rgba(20,19,17,.82); border-radius: 24px; box-shadow: 0 34px 120px rgba(0,0,0,.42); overflow: hidden; position: relative; backdrop-filter: blur(8px); }
    .panel::before { content: "EXTRACT"; position: absolute; right: 34px; top: 38px; color: rgba(255,255,255,.035); font-size: clamp(54px, 12vw, 132px); line-height: .8; font-weight: 950; letter-spacing: -.09em; pointer-events: none; }
    .panel::after { content: ""; position: absolute; left: 0; right: 0; top: 0; height: 1px; background: linear-gradient(90deg, transparent, rgba(255,255,255,.18), transparent); }
    .hero { padding: clamp(28px, 5.5vw, 54px) clamp(26px, 6vw, 64px) 24px; position: relative; }
    .kicker { color: var(--text-tertiary); font-size: 12px; font-weight: 800; margin-bottom: 14px; letter-spacing: .22em; text-transform: uppercase; display: inline-flex; align-items: center; gap: 10px; }
    .kicker::before { content: ""; width: 22px; height: 1px; background: linear-gradient(90deg, var(--accent-b), transparent); }
    h1 { margin: 0; max-width: 680px; color: var(--text-primary); font-size: clamp(38px, 7vw, 72px); line-height: .98; letter-spacing: -.06em; font-weight: 950; }
    h1 .hl { background: linear-gradient(135deg, var(--accent-a) 0%, var(--accent-b) 50%, var(--accent-c) 100%); -webkit-background-clip: text; background-clip: text; -webkit-text-fill-color: transparent; }
    .desc { max-width: 620px; color: var(--text-secondary); margin: 22px 0 0; font-size: 16px; line-height: 1.75; }
    .meta-row { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 22px; }
    .chip { display: inline-flex; align-items: center; gap: 6px; padding: 6px 11px; border: 1px solid var(--border); background: rgba(14,13,12,.6); border-radius: 999px; color: var(--text-tertiary); font-size: 12px; font-weight: 600; }
    .chip svg { width: 13px; height: 13px; opacity: .8; }
    .form { border-top: 1px solid var(--border); background: rgba(31,29,26,.64); padding: clamp(22px, 4.5vw, 38px) clamp(26px, 6vw, 64px) clamp(28px, 4.5vw, 44px); }
    .label-row { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; margin: 0 0 10px; }
    label { color: var(--text-secondary); display: block; font-size: 14px; font-weight: 800; }
    .hint { color: var(--text-tertiary); font-size: 12px; }
    .input-row { display: flex; flex-direction: column; gap: 14px; align-items: stretch; }
    textarea { width: 100%; min-height: 112px; border: 1px solid var(--border-strong); background: rgba(14,13,12,.86); color: var(--text-primary); border-radius: 15px; outline: none; padding: 14px 18px; font: inherit; font-size: 15px; line-height: 1.55; resize: vertical; transition: border-color .16s, box-shadow .16s, background .16s; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; letter-spacing: .01em; }
    textarea::placeholder { color: rgba(185,178,170,.5); font-family: inherit; }
    textarea:focus { border-color: var(--primary); box-shadow: 0 0 0 4px rgba(139,134,128,.16); background: rgba(14,13,12,.96); }
    .format-panel { border: 1px solid var(--border); background: linear-gradient(180deg, rgba(14,13,12,.58), rgba(14,13,12,.36)); border-radius: 16px; padding: 14px; }
    .format-heading { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; color: var(--text-secondary); font-size: 13px; font-weight: 850; margin-bottom: 10px; }
    .format-heading small { color: var(--text-tertiary); font-size: 12px; font-weight: 650; }
    .format-options { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
    .format-card { position: relative; display: block; border: 1px solid var(--border); background: rgba(20,19,17,.68); border-radius: 14px; padding: 10px 15px; min-height: 52px; cursor: pointer; transition: border-color .16s, background .16s, transform .16s, box-shadow .16s; }
    .format-card:hover { transform: translateY(-1px); border-color: rgba(255,255,255,.18); background: rgba(24,23,21,.88); }
    .format-card.active { border-color: rgba(185,240,110,.58); background: linear-gradient(135deg, rgba(255,243,77,.10), rgba(142,215,239,.08)); box-shadow: inset 0 1px 0 rgba(255,255,255,.08); }
    .format-card input { position: absolute; opacity: 0; pointer-events: none; }
    .format-card .format-name { display: block; color: var(--text-primary); font-size: 14px; font-weight: 900; margin-bottom: 7px; }
    .format-card .format-desc { display: block; color: var(--text-tertiary); font-size: 12px; line-height: 1.6; padding-right: 48px; }
    .format-card.active::after { content: "已选择"; position: absolute; right: 12px; top: 10px; color: #1b1a16; background: linear-gradient(135deg, var(--accent-a), var(--accent-b)); border-radius: 999px; padding: 3px 8px; font-size: 11px; font-weight: 900; }
    button { height: 54px; border: 1px solid rgba(255,255,255,.14); background: #f5f2ee; color: #171512; border-radius: 15px; cursor: pointer; padding: 0 26px; font: inherit; font-weight: 900; white-space: nowrap; transition: transform .16s, box-shadow .16s, opacity .16s; display: inline-flex; align-items: center; gap: 8px; letter-spacing: .01em; align-self: center; }
    button svg { width: 16px; height: 16px; }
    button:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 18px 32px rgba(245,242,238,.12); }
    button:disabled { opacity: .58; cursor: wait; }
    .status { min-height: 24px; color: var(--text-tertiary); margin-top: 18px; font-size: 14px; line-height: 1.6; display: flex; align-items: center; gap: 8px; }
    .status::before { content: ""; width: 6px; height: 6px; border-radius: 50%; background: currentColor; opacity: .55; flex: none; }
    .status.ok { color: #86efac; }
    .status.error { color: var(--error); }
    .note { border: 1px solid var(--border); background: rgba(14,13,12,.48); color: var(--text-tertiary); border-radius: 14px; margin-top: 18px; padding: 14px 16px 14px 42px; font-size: 13px; line-height: 1.7; position: relative; }
    .note::before { content: "i"; position: absolute; left: 14px; top: 13px; width: 18px; height: 18px; border-radius: 50%; border: 1px solid rgba(255,255,255,.18); display: grid; place-items: center; font-size: 11px; font-weight: 800; font-style: italic; color: var(--text-secondary); font-family: Georgia, serif; }
    .footer { padding: 18px clamp(20px, 5vw, 72px) 28px; color: var(--text-tertiary); font-size: 12px; display: flex; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
    .footer .left { letter-spacing: .04em; }
    .footer .right { letter-spacing: .12em; text-transform: uppercase; opacity: .7; }
    @media (max-width: 720px) {
      .topbar { padding: 22px 18px; }
      .brand .sub { display: none; }
      .pill { padding: 8px 12px; font-size: 12px; }
      .input-row { grid-template-columns: 1fr; }
      .format-options { grid-template-columns: 1fr; }
      button { width: 100%; justify-content: center; }
      .panel::before { display: none; }
      .footer { padding: 12px 18px 22px; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <span class="logo" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3 3 8l9 13 9-13-9-5Z"/><path d="M12 7v10"/><path d="M7.5 9.5h9"/><path d="m8 14 4 3 4-3"/></svg></span>
        <span class="name">CODEX EXTRACT</span>
      </div>
      <div class="pill"><span class="dot"></span><span>服务在线</span></div>
    </header>
    <main class="center">
      <section class="panel">
        <div class="hero">
          <div class="kicker">Codex Auth File</div>
          <h1>输入卡密，<span class="hl">一键提取</span></h1>
          <p class="desc">支持粘贴卡密或邮箱---keycode 链接，系统验活通过后可导出 CPA ZIP 或 SUB JSON。</p>
          <div class="meta-row">
            <span class="chip"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/></svg>验活后下发</span>
            <span class="chip"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/></svg>CPA ZIP</span>
            <span class="chip"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 7h16"/><path d="M4 12h16"/><path d="M4 17h16"/></svg>SUB JSON</span>
            <span class="chip"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>批量支持</span>
          </div>
        </div>
        <div class="form">
          <div class="label-row">
            <label for="cardCode">卡密 / 提取链接</label>
          </div>
          <div class="input-row">
            <textarea id="cardCode" autocomplete="one-time-code" spellcheck="false" rows="3" placeholder="user@example.com---https://mail.lucker.cc.cd/keycode?email=user@example.com&amp;key=et_xxxxxxxxxxxxxxxxxxxxx&#10;CDX-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"></textarea>
            <div class="format-panel" id="formatPanel">
              <div class="format-heading"><span>提取格式转换</span><small>点击提取后按选中格式下载</small></div>
              <div class="format-options" role="radiogroup" aria-label="提取格式">
                <label class="format-card active" data-format-card="cpa">
                  <input type="radio" name="extractFormat" value="cpa" checked>
                  <span class="format-name">CPA 格式</span>
                </label>
                <label class="format-card" data-format-card="sub">
                  <input type="radio" name="extractFormat" value="sub">
                  <span class="format-name">SUB 格式</span>
                </label>
              </div>
            </div>
            <button id="extractButton" type="button">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/></svg>
              <span>提取</span>
            </button>
          </div>
          <div id="status" class="status">就绪</div>
          <div class="note">每行输入一张卡密或一个邮箱---keycode 链接；CPA 会合并为 ZIP，SUB 会导出单个 JSON。</div>
        </div>
      </section>
    </main>
    <footer class="footer">
      <span class="left">© CODEX EXTRACT</span>
      <span class="right">Codex Auth Pipeline</span>
    </footer>
  </div>
  <script>
    var input = document.getElementById('cardCode');
    var button = document.getElementById('extractButton');
    var statusLine = document.getElementById('status');
    var formatInputs = document.querySelectorAll('input[name="extractFormat"]');
    var formatCards = document.querySelectorAll('[data-format-card]');

    function setStatus(message, type) {
      statusLine.textContent = message || '';
      statusLine.className = 'status' + (type ? ' ' + type : '');
    }

    function filenameFromDisposition(value) {
      if (!value) return '';
      var match = value.match(/filename="?([^";]+)"?/i);
      return match ? match[1] : '';
    }

    function downloadBlob(blob, filename) {
      var url = URL.createObjectURL(blob);
      var a = document.createElement('a');
      a.href = url;
      a.download = filename || 'codex-auth-file.zip';
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
    }

    function defaultDownloadName(format, count) {
      if (format === 'sub') {
        return 'sub2api-account.json';
      }
      return count > 1 ? 'codex-auth-files.zip' : 'codex-auth-file.zip';
    }

    function getSelectedFormat() {
      for (var i = 0; i < formatInputs.length; i++) {
        if (formatInputs[i].checked) return formatInputs[i].value || 'cpa';
      }
      return 'cpa';
    }

    function refreshFormatCards() {
      var format = getSelectedFormat();
      for (var i = 0; i < formatCards.length; i++) {
        var card = formatCards[i];
        var active = card.getAttribute('data-format-card') === format;
        card.className = active ? 'format-card active' : 'format-card';
      }
    }

    async function readError(resp) {
      var type = resp.headers.get('content-type') || '';
      if (type.indexOf('application/json') >= 0) {
        try { return await resp.json(); } catch (errJSON) {}
      }
      return { error: await resp.text() };
    }

    function extractCardCodeInput(value) {
      var trimmed = String(value || '').trim();
      if (!trimmed) return '';
      var candidates = cardCodeInputCandidates(trimmed);
      for (var i = 0; i < candidates.length; i++) {
        var candidate = candidates[i];
        var key = extractCardCodeKeyParam(candidate);
        if (key) return key;
      }
      return trimmed;
    }

    function cardCodeInputCandidates(trimmed) {
      var candidates = [trimmed];
      var markerIndex = trimmed.indexOf('---');
      if (markerIndex >= 0) {
        var suffix = trimmed.slice(markerIndex + 3).trim();
        if (suffix && suffix !== trimmed) candidates.unshift(suffix);
      }
      return candidates;
    }

    function extractCardCodeKeyParam(value) {
      try {
        var parsed = new URL(value, window.location.origin);
        var key = parsed.searchParams.get('key');
        if (key && key.trim()) return key.trim();
      } catch (errParse) {}
      var match = String(value || '').match(/(?:^|[?&#])key=([^&#\s]+)/i);
      if (match && match[1]) {
        try {
          return decodeURIComponent(match[1].replace(/\+/g, ' ')).trim();
        } catch (errDecode) {
          return match[1].trim();
        }
      }
      return '';
    }

    function getCardCodes() {
      return String(input.value || '')
        .replace(/\r\n/g, '\n')
        .replace(/\r/g, '\n')
        .split('\n')
        .map(extractCardCodeInput)
        .filter(Boolean);
    }

    async function extract() {
      var codes = getCardCodes();
      if (codes.length === 0) {
        setStatus('请先输入卡密或提取链接', 'error');
        input.focus();
        return;
      }
      button.disabled = true;
      var format = getSelectedFormat();
      var formatLabel = format === 'sub' ? 'SUB JSON' : 'CPA ZIP';
      setStatus((codes.length > 1 ? '验活中（' + codes.length + '）' : '验活中') + ' · 准备 ' + formatLabel + '…', '');
      try {
        var resp = await fetch('/v0/codex-extract', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ items: codes, format: format })
        });
        if (!resp.ok) {
          var err = await readError(resp);
          throw new Error(err.error || '提取失败');
        }
        var blob = await resp.blob();
        var filename = filenameFromDisposition(resp.headers.get('content-disposition')) || defaultDownloadName(format, codes.length);
        downloadBlob(blob, filename);
        setStatus('提取成功 · ' + formatLabel + ' 已开始下载', 'ok');
        input.value = '';
      } catch (err) {
        setStatus(err.message || String(err), 'error');
      } finally {
        button.disabled = false;
      }
    }

    button.addEventListener('click', extract);
    for (var i = 0; i < formatInputs.length; i++) {
      formatInputs[i].addEventListener('change', refreshFormatCards);
    }
    refreshFormatCards();
    input.addEventListener('keydown', function (event) {
      if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') extract();
    });
  </script>
</body>
</html>`
