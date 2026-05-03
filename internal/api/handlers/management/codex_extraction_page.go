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
  <title>Codex 认证文件提取</title>
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
        radial-gradient(circle at 88% 14%, rgba(255,255,255,.06), transparent 28%),
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
    .shell { min-height: 100vh; display: grid; grid-template-rows: auto 1fr; position: relative; z-index: 1; }
    .topbar { display: flex; align-items: center; justify-content: space-between; padding: 28px clamp(20px, 5vw, 72px); }
    .brand { display: inline-flex; align-items: center; gap: 12px; color: var(--text-primary); font-size: 26px; font-weight: 900; letter-spacing: -.03em; }
    .logo { width: 38px; height: 38px; border-radius: 9px; background: linear-gradient(135deg, #fff34d 0%, #b9f06e 42%, #8ed7ef 100%); box-shadow: 0 14px 40px rgba(141,215,239,.16); display: grid; place-items: center; color: #35312d; }
    .logo svg { width: 24px; height: 24px; }
    .pill { border: 1px solid var(--border); background: rgba(31,29,26,.70); border-radius: 999px; padding: 10px 14px; color: var(--text-secondary); font-size: 13px; display: inline-flex; align-items: center; gap: 8px; }
    .dot { width: 9px; height: 9px; background: var(--success); border-radius: 50%; box-shadow: 0 0 0 5px rgba(16,185,129,.12); }
    .center { display: grid; place-items: center; padding: 24px clamp(18px, 5vw, 72px) 72px; }
    .panel { width: min(100%, 920px); border: 1px solid var(--border); background: rgba(20,19,17,.82); border-radius: 22px; box-shadow: 0 34px 120px rgba(0,0,0,.42); overflow: hidden; position: relative; }
    .panel::before { content: "EXTRACT"; position: absolute; right: 34px; top: 42px; color: rgba(255,255,255,.035); font-size: clamp(54px, 12vw, 132px); line-height: .8; font-weight: 950; letter-spacing: -.09em; pointer-events: none; }
    .hero { padding: clamp(34px, 6vw, 58px) clamp(26px, 6vw, 64px) 24px; position: relative; }
    .kicker { color: var(--text-tertiary); font-size: 14px; font-weight: 800; margin-bottom: 10px; }
    h1 { margin: 0; max-width: 650px; color: var(--text-primary); font-size: clamp(38px, 7vw, 72px); line-height: .96; letter-spacing: -.07em; font-weight: 950; }
    .desc { max-width: 600px; color: var(--text-secondary); margin: 18px 0 0; font-size: 16px; line-height: 1.7; }
    .form { border-top: 1px solid var(--border); background: rgba(31,29,26,.64); padding: clamp(26px, 5vw, 42px) clamp(26px, 6vw, 64px) clamp(32px, 5vw, 48px); }
    label { color: var(--text-secondary); display: block; margin: 0 0 10px; font-size: 14px; font-weight: 800; }
    .input-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 12px; align-items: start; }
    textarea { width: 100%; min-height: 132px; border: 1px solid var(--border-strong); background: rgba(14,13,12,.86); color: var(--text-primary); border-radius: 15px; outline: none; padding: 15px 17px; font: inherit; font-size: 16px; line-height: 1.55; resize: vertical; transition: border-color .16s, box-shadow .16s, background .16s; }
    textarea::placeholder { color: rgba(185,178,170,.56); }
    textarea:focus { border-color: var(--primary); box-shadow: 0 0 0 4px rgba(139,134,128,.16); background: rgba(14,13,12,.96); }
    button { height: 54px; border: 1px solid rgba(255,255,255,.14); background: #f5f2ee; color: #171512; border-radius: 15px; cursor: pointer; padding: 0 24px; font: inherit; font-weight: 900; white-space: nowrap; transition: transform .16s, box-shadow .16s, opacity .16s; }
    button:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 18px 32px rgba(245,242,238,.12); }
    button:disabled { opacity: .58; cursor: wait; }
    .status { min-height: 24px; color: var(--text-tertiary); margin-top: 16px; font-size: 14px; line-height: 1.6; }
    .status.ok { color: #86efac; }
    .status.error { color: var(--error); }
    .note { border: 1px solid var(--border); background: rgba(14,13,12,.48); color: var(--text-tertiary); border-radius: 14px; margin-top: 18px; padding: 14px 16px; font-size: 13px; line-height: 1.65; }
    @media (max-width: 720px) {
      .topbar { padding: 22px 18px; }
      .pill { display: none; }
      .input-row { grid-template-columns: 1fr; }
      button { width: 100%; }
      .panel::before { display: none; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <header class="topbar">
      <div class="brand">
        <span class="logo" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="M12 3 3 8l9 13 9-13-9-5Z"/><path d="M12 7v10"/><path d="M7.5 9.5h9"/><path d="m8 14 4 3 4-3"/></svg></span>
        <span>CPAMC</span>
      </div>
      <div class="pill"><span class="dot"></span><span>Codex 认证文件提取</span></div>
    </header>
    <main class="center">
      <section class="panel">
        <div class="hero">
          <div class="kicker">Codex Auth File</div>
          <h1>输入卡密，提取认证文件</h1>
          <p class="desc">系统会随机选择一个可用的 Codex 认证文件并先进行验活；验活通过后自动打包为 ZIP 下载。</p>
        </div>
        <div class="form">
          <label for="cardCode">卡密（一行一个，支持批量）</label>
          <div class="input-row">
            <textarea id="cardCode" autocomplete="one-time-code" spellcheck="false" rows="4" placeholder="请输入你的卡密；多行可批量提取&#10;CDX-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX&#10;CDX-YYYYYYYYYYYYYYYYYYYYYYYYYYYYYYYY"></textarea>
            <button id="extractButton" type="button">提取文件</button>
          </div>
          <div id="status" class="status">等待输入卡密；支持一行一个卡密批量提取。</div>
          <div class="note">一个卡密只能提取一个 Codex 认证 JSON 文件；批量提交会按卡密数量提取多个认证文件并打包为同一个 ZIP 下载。若当前随机账号状态异常，系统会自动更换其他账号继续验活。</div>
        </div>
      </section>
    </main>
  </div>
  <script>
    var input = document.getElementById('cardCode');
    var button = document.getElementById('extractButton');
    var statusLine = document.getElementById('status');

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

    async function readError(resp) {
      var type = resp.headers.get('content-type') || '';
      if (type.indexOf('application/json') >= 0) {
        try { return await resp.json(); } catch (_) {}
      }
      return { error: await resp.text() };
    }

    function getCardCodes() {
      return input.value
        .split(/\r?\n/)
        .map(function (item) { return item.trim(); })
        .filter(Boolean);
    }

    async function extract() {
      var codes = getCardCodes();
      if (codes.length === 0) {
        setStatus('请先输入卡密。', 'error');
        input.focus();
        return;
      }
      button.disabled = true;
      setStatus(codes.length > 1 ? '正在批量验活 ' + codes.length + ' 个卡密并准备下载，请稍候...' : '正在验活并准备下载，请稍候...', '');
      try {
        var resp = await fetch('/v0/codex-extract', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ items: codes })
        });
        if (!resp.ok) {
          var err = await readError(resp);
          throw new Error(err.error || '提取失败');
        }
        var blob = await resp.blob();
        var filename = filenameFromDisposition(resp.headers.get('content-disposition')) || (codes.length > 1 ? 'codex-auth-files.zip' : 'codex-auth-file.zip');
        downloadBlob(blob, filename);
        setStatus(codes.length > 1 ? '批量提取成功，ZIP 已开始下载。' : '提取成功，ZIP 已开始下载。', 'ok');
        input.value = '';
      } catch (err) {
        setStatus(err.message || String(err), 'error');
      } finally {
        button.disabled = false;
      }
    }

    button.addEventListener('click', extract);
    input.addEventListener('keydown', function (event) {
      if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') extract();
    });
  </script>
</body>
</html>`
