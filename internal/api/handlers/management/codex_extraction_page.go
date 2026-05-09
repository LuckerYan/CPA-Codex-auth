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
    .center .panel { transform: translateY(clamp(-30px, -2.2vw, -18px)); }
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
    .result-modal { --modal-enter-ease: cubic-bezier(.22, 1, .36, 1); --modal-exit-ease: cubic-bezier(.22, 1, .36, 1); position: fixed; inset: 0; z-index: 50; display: grid; place-items: center; padding: 22px; background: rgba(5,5,4,.56); opacity: 0; visibility: hidden; pointer-events: none; transition: opacity .24s var(--modal-exit-ease), visibility 0s linear .24s; }
    .result-modal:not([hidden]) { will-change: opacity; }
    .result-modal.is-open { opacity: 1; visibility: visible; pointer-events: auto; transition: opacity .24s var(--modal-enter-ease), visibility 0s; }
    .result-modal.is-closing { opacity: 0; visibility: visible; pointer-events: none; transition: opacity .2s var(--modal-exit-ease), visibility 0s linear .2s; }
    .result-modal[hidden] { display: none; }
    .result-card { width: min(100%, 620px); max-height: min(82vh, 720px); overflow: auto; scrollbar-gutter: stable; border: 1px solid rgba(255,255,255,.14); border-radius: 22px; background: linear-gradient(180deg, rgba(31,29,26,.97), rgba(14,13,12,.95)); box-shadow: 0 20px 56px rgba(0,0,0,.36), inset 0 1px 0 rgba(255,255,255,.06); padding: 22px; opacity: 0; transform: translate3d(0, 20px, 0) scale(.96); transform-origin: center; transition: transform .24s var(--modal-enter-ease), opacity .24s ease-out; backface-visibility: hidden; contain: paint; }
    .result-modal:not([hidden]) .result-card { will-change: transform, opacity; }
    .result-modal.is-open .result-card { opacity: 1; transform: translate3d(0, 0, 0) scale(1); }
    .result-modal.is-closing .result-card { opacity: 0; transform: translate3d(0, 12px, 0) scale(.985); transition: transform .2s var(--modal-exit-ease), opacity .18s ease-out; }
    @media (prefers-reduced-motion: reduce) {
      .result-modal, .result-modal.is-open, .result-modal.is-closing, .result-card, .result-modal.is-open .result-card, .result-modal.is-closing .result-card { transition-duration: .01ms !important; transform: none !important; }
    }
    .result-title { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-bottom: 14px; color: var(--text-primary); font-size: 18px; font-weight: 950; }
    .result-close { width: 34px; height: 34px; min-width: 34px; padding: 0; justify-content: center; border-radius: 999px; background: rgba(255,255,255,.06); color: var(--text-secondary); }
    .result-counts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; margin: 14px 0; }
    .result-count { border: 1px solid var(--border); border-radius: 15px; background: rgba(14,13,12,.55); padding: 13px 14px; }
    .result-count span { display: block; color: var(--text-tertiary); font-size: 12px; font-weight: 800; margin-bottom: 5px; }
    .result-count strong { color: var(--text-primary); font-size: 24px; line-height: 1; }
    .result-count.success strong { color: #86efac; }
    .result-count.failed strong { color: var(--error); }
    .result-help { margin: 0 0 12px; color: var(--text-tertiary); font-size: 13px; line-height: 1.7; }
    .failure-group { border: 1px solid rgba(239,154,139,.22); border-radius: 15px; background: rgba(239,154,139,.055); padding: 13px 14px; margin-top: 10px; }
    .failure-group-title { color: #ffd0c8; font-size: 13px; font-weight: 900; margin-bottom: 8px; }
    .failure-code-list { margin: 0; padding: 0; list-style: none; display: grid; gap: 6px; }
    .failure-code-list li { color: var(--text-secondary); font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; line-height: 1.45; overflow-wrap: anywhere; }
    .progress-shell { position: fixed; left: 50%; top: 50%; bottom: auto; z-index: 30; width: min(calc(100vw - 36px), 680px); display: grid; gap: 10px; padding: 14px 16px 15px; border: 1px solid rgba(255,255,255,.14); border-radius: 18px; background: linear-gradient(180deg, rgba(31,29,26,.92), rgba(14,13,12,.88)); box-shadow: 0 28px 78px rgba(0,0,0,.48), inset 0 1px 0 rgba(255,255,255,.06); backdrop-filter: blur(14px); pointer-events: none; transform: translate(-50%, -50%); }
    .progress-shell[hidden] { display: none; }
    .progress-shell::before { content: ""; position: absolute; left: 18px; right: 18px; top: 0; height: 1px; background: linear-gradient(90deg, transparent, rgba(185,240,110,.62), rgba(142,215,239,.32), transparent); }
    .progress-meta { display: flex; align-items: center; justify-content: space-between; gap: 12px; color: var(--text-tertiary); font-size: 12px; font-weight: 750; letter-spacing: .08em; text-transform: uppercase; }
    .progress-meta .stage { color: var(--text-secondary); font-size: 13px; font-weight: 700; letter-spacing: .02em; text-transform: none; }
    .progress-track { position: relative; height: 10px; border: 1px solid rgba(255,255,255,.12); border-radius: 999px; background: linear-gradient(180deg, rgba(255,255,255,.045), rgba(255,255,255,.014)); overflow: hidden; box-shadow: inset 0 1px 0 rgba(255,255,255,.03), 0 14px 30px rgba(0,0,0,.18); }
    .progress-fill { position: absolute; inset: 1px auto 1px 1px; width: 0%; border-radius: inherit; background: linear-gradient(90deg, var(--accent-a) 0%, var(--accent-b) 48%, var(--accent-c) 100%); box-shadow: 0 0 18px rgba(185,240,110,.24), 0 0 30px rgba(142,215,239,.10); transition: width .24s ease, background .24s ease, box-shadow .24s ease; overflow: hidden; }
    .progress-fill::after { content: ""; position: absolute; inset: 0; background: linear-gradient(110deg, transparent 22%, rgba(255,255,255,.28) 42%, transparent 62%); transform: translateX(-60%); animation: progressShimmer 1.8s linear infinite; }
    .progress-shell.success .progress-fill { background: linear-gradient(90deg, var(--accent-a) 0%, var(--accent-b) 48%, var(--accent-c) 100%); }
    .progress-shell.error .progress-fill { background: linear-gradient(90deg, #ef9a8b 0%, #f4c76a 100%); box-shadow: 0 0 16px rgba(239,154,139,.18), 0 0 22px rgba(244,199,106,.10); }
    @keyframes progressShimmer { from { transform: translateX(-60%); } to { transform: translateX(60%); } }
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
      .center .panel { transform: none; }
      .progress-shell { top: 50%; bottom: auto; width: calc(100vw - 24px); border-radius: 16px; padding: 12px 13px 13px; }
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
          <div class="note">每行输入一张卡密或一个邮箱---keycode 链接；CPA 会合并为 ZIP，SUB 会导出单个 JSON。</div>
        </div>
      </section>
    </main>
    <div id="progressShell" class="progress-shell" hidden aria-live="polite" aria-atomic="true">
      <div class="progress-meta">
        <span id="progressStage" class="stage">等待提取开始</span>
        <span id="progressPercent">0%</span>
      </div>
      <div class="progress-track" role="progressbar" aria-label="提取进度" aria-valuemin="0" aria-valuemax="100" aria-valuenow="0">
        <div class="progress-fill" id="progressFill"></div>
      </div>
    </div>
    <div id="resultModal" class="result-modal" hidden role="dialog" aria-modal="true" aria-labelledby="resultTitle">
      <div class="result-card">
        <div class="result-title">
          <span id="resultTitle">提取结果</span>
          <button id="resultCloseButton" class="result-close" type="button" aria-label="关闭">×</button>
        </div>
        <div class="result-counts">
          <div class="result-count success"><span>成功个数</span><strong id="resultSuccessCount">0</strong></div>
          <div class="result-count failed"><span>失败个数</span><strong id="resultFailedCount">0</strong></div>
        </div>
        <p id="resultHelp" class="result-help">提取完成。</p>
        <div id="resultFailureGroups"></div>
      </div>
    </div>
    <footer class="footer">
      <span class="left">© CODEX EXTRACT</span>
      <span class="right">Codex Auth Pipeline</span>
    </footer>
  </div>
  <script>
    var input = document.getElementById('cardCode');
    var button = document.getElementById('extractButton');
    var formatInputs = document.querySelectorAll('input[name="extractFormat"]');
    var formatCards = document.querySelectorAll('[data-format-card]');
    var progressShell = document.getElementById('progressShell');
    var progressTrack = progressShell ? progressShell.querySelector('.progress-track') : null;
    var progressFill = document.getElementById('progressFill');
    var progressStage = document.getElementById('progressStage');
    var progressPercent = document.getElementById('progressPercent');
    var resultModal = document.getElementById('resultModal');
    var resultCard = resultModal ? resultModal.querySelector('.result-card') : null;
    var resultTitle = document.getElementById('resultTitle');
    var resultSuccessCount = document.getElementById('resultSuccessCount');
    var resultFailedCount = document.getElementById('resultFailedCount');
    var resultHelp = document.getElementById('resultHelp');
    var resultFailureGroups = document.getElementById('resultFailureGroups');
    var resultCloseButton = document.getElementById('resultCloseButton');
    var progressTimer = null;
    var progressResetTimer = null;
    var resultHideTimer = null;
    var resultHideTransitionEnd = null;
    var resultShowFrame = null;
    var progressValue = 0;
    var progressTarget = 0;
    var buttonLabel = button.querySelector('span');
    var buttonDefaultLabel = buttonLabel ? buttonLabel.textContent : '提取';

    function setButtonBusy(isBusy) {
      if (buttonLabel) {
        buttonLabel.textContent = isBusy ? '提取中…' : buttonDefaultLabel;
      }
      button.setAttribute('aria-busy', isBusy ? 'true' : 'false');
    }

    function stopProgressTimers() {
      if (progressTimer) {
        clearInterval(progressTimer);
        progressTimer = null;
      }
      if (progressResetTimer) {
        clearTimeout(progressResetTimer);
        progressResetTimer = null;
      }
    }

    function hideProgress() {
      stopProgressTimers();
      progressValue = 0;
      progressTarget = 0;
      if (progressShell && progressTrack && progressFill && progressStage && progressPercent) {
        progressShell.hidden = true;
        progressShell.className = 'progress-shell';
        progressTrack.setAttribute('aria-valuenow', '0');
        progressFill.style.width = '0%';
        progressStage.textContent = '等待提取开始';
        progressPercent.textContent = '0%';
      }
    }

    function renderProgress(value, stage, variant) {
      if (!(progressShell && progressTrack && progressFill && progressStage && progressPercent)) return;
      var bounded = Math.max(0, Math.min(100, Math.round(value || 0)));
      progressShell.hidden = false;
      progressShell.className = 'progress-shell' + (variant ? ' ' + variant : '');
      progressTrack.setAttribute('aria-valuenow', String(bounded));
      progressFill.style.width = bounded + '%';
      progressPercent.textContent = bounded + '%';
      if (typeof stage === 'string') {
        progressStage.textContent = stage;
      }
    }

    function getExtractConcurrency() {
      return 10;
    }

    function startProgress(totalCount, formatLabel, concurrency) {
      if (!(progressShell && progressTrack && progressFill && progressStage && progressPercent)) return;
      stopProgressTimers();
      progressValue = 8;
      progressTarget = totalCount > 6 ? 82 : 88;
      var concurrencyText = concurrency > 1 ? ' · 并发 ' + concurrency : '';
      renderProgress(progressValue, totalCount > 1 ? '验活中 · ' + totalCount + ' 项' + concurrencyText + ' · 准备 ' + formatLabel + '…' : '验活中' + concurrencyText + ' · 准备 ' + formatLabel + '…', 'busy');
      progressTimer = setInterval(function () {
        if (progressValue >= progressTarget) return;
        var step = progressValue < 30 ? 6 : progressValue < 70 ? 3 : 1;
        if (totalCount > 3 && progressValue < 58) step += 1;
        progressValue = Math.min(progressTarget, progressValue + step);
        renderProgress(progressValue, undefined, 'busy');
      }, 170);
    }

    function completeProgress(formatLabel) {
      if (!(progressShell && progressTrack && progressFill && progressStage && progressPercent)) return;
      stopProgressTimers();
      renderProgress(100, formatLabel + ' 已完成', 'success');
      progressResetTimer = setTimeout(hideProgress, 900);
    }

    function failProgress(message) {
      if (!(progressShell && progressTrack && progressFill && progressStage && progressPercent)) return;
      stopProgressTimers();
      renderProgress(100, message || '提取失败', 'error');
      progressResetTimer = setTimeout(hideProgress, 1400);
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

    function blobFromBase64(encoded, contentType) {
      var binary = atob(String(encoded || ''));
      var chunkSize = 8192;
      var chunks = [];
      for (var offset = 0; offset < binary.length; offset += chunkSize) {
        var slice = binary.slice(offset, offset + chunkSize);
        var bytes = new Uint8Array(slice.length);
        for (var i = 0; i < slice.length; i++) bytes[i] = slice.charCodeAt(i);
        chunks.push(bytes);
      }
      return new Blob(chunks, { type: contentType || 'application/octet-stream' });
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

    function parseExtractSummaryHeader(resp) {
      var encoded = resp && resp.headers ? resp.headers.get('x-codex-extract-summary') : '';
      if (!encoded) return null;
      try {
        var binary = atob(encoded);
        var text = '';
        if (window.TextDecoder) {
          var bytes = new Uint8Array(binary.length);
          for (var i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
          text = new TextDecoder('utf-8').decode(bytes);
        } else {
          text = decodeURIComponent(escape(binary));
        }
        return JSON.parse(text);
      } catch (errSummary) {
        return null;
      }
    }

    function fallbackSummary(codes, format) {
      return {
        status: 'ok',
        requested: codes.length,
        success: codes.length,
        failed: 0,
        format: format,
        failure_groups: []
      };
    }

    function cancelResultShowFrame() {
      if (!resultShowFrame) return;
      cancelAnimationFrame(resultShowFrame);
      resultShowFrame = null;
    }

    function clearResultHideHandler() {
      if (resultHideTimer) {
        clearTimeout(resultHideTimer);
        resultHideTimer = null;
      }
      if (resultCard && resultHideTransitionEnd) {
        resultCard.removeEventListener('transitionend', resultHideTransitionEnd);
        resultHideTransitionEnd = null;
      }
    }

    function finishResultModalClose() {
      if (!resultModal) return;
      clearResultHideHandler();
      resultModal.hidden = true;
      resultModal.classList.remove('is-open', 'is-closing');
    }

    function closeResultModal() {
      if (!resultModal || resultModal.hidden) return;
      cancelResultShowFrame();
      clearResultHideHandler();
      resultModal.classList.remove('is-open');
      resultModal.classList.add('is-closing');
      if (resultCard) {
        resultHideTransitionEnd = function (event) {
          if (event.target !== resultCard || event.propertyName !== 'transform') return;
          finishResultModalClose();
        };
        resultCard.addEventListener('transitionend', resultHideTransitionEnd);
      }
      resultHideTimer = setTimeout(finishResultModalClose, 300);
    }

    function showResultModal(summary, formatLabel, message) {
      if (!(resultModal && resultTitle && resultSuccessCount && resultFailedCount && resultHelp && resultFailureGroups)) return;
      summary = summary || {};
      var status = String(summary.status || '').toLowerCase();
      var success = Number(summary.success || 0);
      var failed = Number(summary.failed || 0);
      var groups = Array.isArray(summary.failure_groups) ? summary.failure_groups : (Array.isArray(summary.failureGroups) ? summary.failureGroups : []);
      if (status === 'partial') {
        resultTitle.textContent = '提取部分成功';
      } else if (status === 'failed') {
        resultTitle.textContent = '提取失败';
      } else {
        resultTitle.textContent = failed > 0 ? (success > 0 ? '提取部分成功' : '提取失败') : '提取成功';
      }
      resultSuccessCount.textContent = String(success);
      resultFailedCount.textContent = String(failed);
      if (message && String(message).trim() !== '') {
        resultHelp.textContent = String(message).trim();
      } else if (failed > 0 && success > 0) {
        resultHelp.textContent = '已为成功的卡密导出 ' + formatLabel + '；失败卡密已按错误原因分组显示。';
      } else if (failed > 0) {
        resultHelp.textContent = '本次没有可导出的卡密；失败卡密已按错误原因分组显示。';
      } else {
        resultHelp.textContent = '全部卡密提取成功，' + formatLabel + ' 已开始下载。';
      }
      resultFailureGroups.innerHTML = '';
      for (var i = 0; i < groups.length; i++) {
        var group = groups[i] || {};
        var codes = Array.isArray(group.codes) ? group.codes : [];
        if (codes.length === 0) continue;
        var box = document.createElement('div');
        box.className = 'failure-group';
        var title = document.createElement('div');
        title.className = 'failure-group-title';
        title.textContent = (group.message || '提取失败') + '（' + codes.length + ' 个）';
        box.appendChild(title);
        var list = document.createElement('ul');
        list.className = 'failure-code-list';
        for (var j = 0; j < codes.length; j++) {
          var item = document.createElement('li');
          item.textContent = codes[j];
          list.appendChild(item);
        }
        box.appendChild(list);
        resultFailureGroups.appendChild(box);
      }
      clearResultHideHandler();
      cancelResultShowFrame();
      if (resultModal.hidden) {
        resultModal.hidden = false;
      }
      resultModal.classList.remove('is-open', 'is-closing');
      resultShowFrame = requestAnimationFrame(function () {
        resultShowFrame = null;
        if (!resultModal || resultModal.hidden) return;
        resultModal.classList.add('is-open');
      });
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
        showResultModal({ status: 'failed', requested: 0, success: 0, failed: 0, format: getSelectedFormat(), failure_groups: [] }, getSelectedFormat() === 'sub' ? 'SUB JSON' : 'CPA ZIP', '请先输入卡密或提取链接');
        input.focus();
        return;
      }
      button.disabled = true;
      setButtonBusy(true);
      var format = getSelectedFormat();
      var formatLabel = format === 'sub' ? 'SUB JSON' : 'CPA ZIP';
      var concurrency = getExtractConcurrency();
      startProgress(codes.length, formatLabel, concurrency);
      try {
        var resp = await fetch('/v0/codex-extract', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ items: codes, format: format, concurrency: concurrency })
        });
        if (!resp.ok) {
          var err = await readError(resp);
          var httpErr = new Error(err.error || '提取失败');
          httpErr.summary = err.summary || null;
          throw httpErr;
        }
        var respType = resp.headers.get('content-type') || '';
        var disposition = resp.headers.get('content-disposition') || '';
        var blob;
        var filename;
        var summary;
        if (respType.indexOf('application/json') >= 0 && !disposition) {
          var payload = await resp.json();
          if (!payload.download_base64) throw new Error(payload.error || '提取失败');
          blob = blobFromBase64(payload.download_base64, payload.content_type);
          filename = payload.download_filename || defaultDownloadName(format, codes.length);
          summary = payload.summary || fallbackSummary(codes, format);
        } else {
          blob = await resp.blob();
          filename = filenameFromDisposition(disposition) || defaultDownloadName(format, codes.length);
          summary = parseExtractSummaryHeader(resp) || fallbackSummary(codes, format);
        }
        downloadBlob(blob, filename);
        completeProgress(formatLabel);
        hideProgress();
        showResultModal(summary, formatLabel);
        if (!summary.failed) input.value = '';
      } catch (err) {
        var errorSummary = err && err.summary ? err.summary : { status: 'failed', requested: codes.length, success: 0, failed: 0, format: format, failure_groups: [] };
        hideProgress();
        showResultModal(errorSummary, formatLabel, err.message || String(err));
      } finally {
        button.disabled = false;
        setButtonBusy(false);
      }
    }

    button.addEventListener('click', extract);
    if (resultCloseButton) resultCloseButton.addEventListener('click', closeResultModal);
    if (resultModal) resultModal.addEventListener('click', function (event) {
      if (event.target === resultModal) closeResultModal();
    });
    for (var i = 0; i < formatInputs.length; i++) {
      formatInputs[i].addEventListener('change', refreshFormatCards);
    }
    refreshFormatCards();
    input.addEventListener('keydown', function (event) {
      if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') extract();
    });
    document.addEventListener('keydown', function (event) {
      if (event.key === 'Escape') closeResultModal();
    });
  </script>
</body>
</html>`
