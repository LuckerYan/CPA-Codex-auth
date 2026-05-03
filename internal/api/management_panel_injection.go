package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func (s *Server) serveManagementControlPanelAsset(c *gin.Context, filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.WithError(err).Error("failed to read management control panel asset")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", injectCodexCardManagementPanel(data))
}

func injectCodexCardManagementPanel(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	if bytes.Contains(data, []byte("codex-card-management-injection")) {
		return data
	}
	marker := []byte("</body>")
	script := []byte(fmt.Sprintf("\n<script id=\"codex-card-management-injection\">\n%s\n</script>\n", codexCardManagementPanelScript))
	idx := bytes.LastIndex(bytes.ToLower(data), marker)
	if idx < 0 {
		out := make([]byte, 0, len(data)+len(script))
		out = append(out, data...)
		out = append(out, script...)
		return out
	}
	out := make([]byte, 0, len(data)+len(script))
	out = append(out, data[:idx]...)
	out = append(out, script...)
	out = append(out, data[idx:]...)
	return out
}

const codexCardManagementPanelScript = `
(function () {
  "use strict";

  var PAGE_HASH = "#/codex-cards";
  var AUTH_KEY = "cli-proxy-auth";
  var SECURE_PREFIX = "enc::v1::";
  var SECURE_NAMESPACE = "cli-proxy-api-webui::secure-storage";
  var ACTIVE_KEY = "codex-card-panel-active";
  var observerStarted = false;
  var lastRenderToken = 0;
  var codexPageActive = window.location.hash === PAGE_HASH;
  var allCards = [];
  var currentCards = [];

  function iconSVG() {
    return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.72" stroke-linecap="square" stroke-linejoin="miter" aria-hidden="true" focusable="false" stroke-miterlimit="10" width="18" height="18"><path d="M4 7a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v3a2 2 0 0 0 0 4v3a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-3a2 2 0 0 0 0-4Z" fill="currentColor" fill-opacity="0.10"></path><path d="M9 8h6"></path><path d="M9 12h6"></path><path d="M9 16h4"></path></svg>';
  }

  function ensureStyles() {
    if (document.getElementById("codex-card-management-style")) return;
    var style = document.createElement("style");
    style.id = "codex-card-management-style";
    style.textContent = ` + "`" + `
.codex-card-admin-page{width:100%;max-width:1120px;margin:0 auto;color:var(--text-primary);position:relative}
.codex-card-admin-page *{box-sizing:border-box}
body.codex-card-admin-active .main-content > :not(.codex-card-admin-page){display:none!important}
.codex-card-admin-page-header{margin:0 0 24px}
.codex-card-admin-title{color:var(--text-primary);letter-spacing:-.02em;margin:0 0 12px;font-size:32px;font-weight:800;line-height:1.25}
.codex-card-admin-desc{color:var(--text-secondary);max-width:none;margin:0;font-size:15px;line-height:1.6}
.codex-card-admin-grid{grid-template-columns:minmax(0,1fr) minmax(0,1fr);gap:18px;display:grid}
.codex-card-admin-card{border:1px solid var(--border-color);background:var(--bg-secondary);border-radius:16px;padding:24px;box-shadow:none}
.codex-card-admin-card.wide{grid-column:1/-1}
.codex-card-admin-card h2{color:var(--text-primary);margin:0 0 8px;font-size:22px;font-weight:800;line-height:1.3}
.codex-card-admin-muted{color:var(--text-secondary);margin:0 0 16px;font-size:13px;line-height:1.6}
.codex-card-admin-label{color:var(--text-secondary);margin:14px 0 8px;font-size:13px;font-weight:700;display:block}
.codex-card-admin-input,.codex-card-admin-textarea{width:100%;border:1px solid var(--border-color);background:var(--bg-primary);color:var(--text-primary);border-radius:10px;outline:none;padding:10px 12px;font:inherit;transition:border-color .15s,box-shadow .15s,background .15s}
.codex-card-admin-textarea{min-height:170px;resize:vertical;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,Liberation Mono,Courier New,monospace;font-size:13px;line-height:1.55}
.codex-card-admin-input:focus,.codex-card-admin-textarea:focus{border-color:var(--primary-color);box-shadow:0 0 0 3px color-mix(in srgb,var(--primary-color) 18%,transparent)}
.codex-card-admin-row{align-items:center;gap:12px;display:flex}
.codex-card-admin-row>*{flex:1}
.codex-card-admin-actions{flex-wrap:wrap;gap:10px;margin-top:14px;display:flex}
.codex-card-admin-button{border:1px solid color-mix(in srgb,var(--primary-color) 60%,var(--border-color));background:var(--primary-color);color:#fff;cursor:pointer;border-radius:10px;align-items:center;justify-content:center;gap:8px;min-height:38px;padding:8px 14px;font:inherit;font-weight:700;transition:transform .15s,box-shadow .15s,opacity .15s;display:inline-flex}
.codex-card-admin-button:hover:not(:disabled){transform:translateY(-1px);box-shadow:0 12px 22px color-mix(in srgb,var(--primary-color) 24%,transparent)}
.codex-card-admin-button:disabled{opacity:.55;cursor:wait}
.codex-card-admin-button.secondary{background:var(--bg-secondary);color:var(--text-primary);border-color:var(--border-color)}
.codex-card-admin-button.danger{background:color-mix(in srgb,var(--error-color) 86%,#111);border-color:color-mix(in srgb,var(--error-color) 65%,var(--border-color));color:#fff}
.codex-card-admin-stats{grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-bottom:18px;display:grid}
.codex-card-admin-stat{border:1px solid var(--border-color);background:var(--bg-secondary);border-radius:14px;padding:16px}
.codex-card-admin-stat-value{color:var(--text-primary);font-size:28px;font-weight:900;line-height:1}
.codex-card-admin-stat-label{color:var(--text-secondary);margin-top:8px;font-size:12px;font-weight:700}
.codex-card-admin-status{color:var(--text-secondary);min-height:22px;margin-top:12px;font-size:13px;line-height:1.5}
.codex-card-admin-status.ok{color:var(--success-color)}
.codex-card-admin-status.error{color:var(--error-color)}
.codex-card-admin-output{white-space:pre-wrap;word-break:break-word;border:1px solid var(--border-color);background:var(--bg-primary);color:var(--text-primary);border-radius:12px;max-height:280px;min-height:90px;margin:14px 0 0;padding:14px;font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,Liberation Mono,Courier New,monospace;font-size:12.5px;line-height:1.55;overflow:auto}
.codex-card-admin-list-head{align-items:flex-start;justify-content:space-between;gap:16px;display:flex}
.codex-card-admin-list-head-text{min-width:0}
.codex-card-admin-table-wrap{border:1px solid var(--border-color);border-radius:14px;overflow:hidden}
.codex-card-admin-table{width:100%;border-collapse:collapse;font-size:13px}
.codex-card-admin-table th,.codex-card-admin-table td{border-bottom:1px solid var(--border-color);padding:11px 12px;text-align:left;vertical-align:top}
.codex-card-admin-table th{color:var(--text-secondary);background:color-mix(in srgb,var(--bg-tertiary) 78%,transparent);font-size:12px;font-weight:800}
.codex-card-admin-table tr:last-child td{border-bottom:0}
.codex-card-admin-table th.select,.codex-card-admin-table td.select{width:48px;text-align:center;vertical-align:middle}
.codex-card-admin-checkbox{appearance:none;width:17px;height:17px;margin:0;border:1px solid var(--border-color);background:var(--bg-primary);border-radius:5px;cursor:pointer;display:inline-grid;place-content:center;transition:background .15s,border-color .15s,box-shadow .15s}
.codex-card-admin-checkbox:checked{background:var(--primary-color);border-color:var(--primary-color)}
.codex-card-admin-checkbox:checked:after{content:"";width:8px;height:5px;border-left:2px solid #fff;border-bottom:2px solid #fff;transform:rotate(-45deg) translate(1px,-1px)}
.codex-card-admin-checkbox:focus-visible{outline:none;box-shadow:0 0 0 3px color-mix(in srgb,var(--primary-color) 20%,transparent)}
.codex-card-admin-code{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,Liberation Mono,Courier New,monospace;font-weight:800}
.codex-card-admin-pill{border:1px solid var(--border-color);border-radius:9999px;padding:3px 9px;font-size:12px;font-weight:800;display:inline-flex}
.codex-card-admin-pill.unused{color:var(--success-color);background:color-mix(in srgb,var(--success-color) 12%,transparent);border-color:color-mix(in srgb,var(--success-color) 35%,var(--border-color))}
.codex-card-admin-pill.redeemed{color:var(--text-secondary);background:color-mix(in srgb,var(--text-secondary) 10%,transparent)}
.codex-card-admin-empty{color:var(--text-secondary);padding:26px;text-align:center}
.codex-card-admin-link{color:var(--text-primary);text-decoration:none;border-bottom:1px solid var(--border-color)}
.codex-card-admin-bulkbar{border:1px solid var(--border-color);background:var(--bg-secondary);border-radius:14px;align-items:center;justify-content:space-between;gap:12px;margin:14px 0 14px;padding:14px;display:flex;flex-wrap:wrap}
.codex-card-admin-search{min-width:260px;flex:1 1 360px}
.codex-card-admin-search .codex-card-admin-input{height:40px}
.codex-card-admin-checklabel{color:var(--text-secondary);align-items:center;gap:9px;font-size:13px;font-weight:800;display:inline-flex;cursor:pointer}
.codex-card-admin-selection{color:var(--text-secondary);font-size:13px;font-weight:700}
.codex-card-admin-bulk-actions{align-items:center;justify-content:flex-end;gap:8px;display:flex;flex:0 0 auto;flex-wrap:wrap}
.codex-card-admin-bulk-actions .codex-card-admin-button{min-height:36px;padding:8px 12px;font-size:13px}
@media (max-width:900px){.codex-card-admin-grid,.codex-card-admin-stats{grid-template-columns:1fr}.codex-card-admin-row,.codex-card-admin-list-head{align-items:stretch;flex-direction:column}.codex-card-admin-bulkbar{align-items:stretch;flex-direction:column}.codex-card-admin-search{min-width:0;flex:auto}.codex-card-admin-bulk-actions{align-items:stretch;flex-direction:column}.codex-card-admin-bulk-actions .codex-card-admin-button{width:100%}}
` + "`" + `;
    document.head.appendChild(style);
  }

  function decodeSecure(raw) {
    if (!raw || !raw.startsWith(SECURE_PREFIX)) return raw;
    try {
      var seed = SECURE_NAMESPACE + "|" + window.location.host + "|" + navigator.userAgent;
      var key = new TextEncoder().encode(seed);
      var bin = atob(raw.slice(SECURE_PREFIX.length));
      var bytes = new Uint8Array(bin.length);
      for (var i = 0; i < bin.length; i += 1) bytes[i] = bin.charCodeAt(i) ^ key[i % key.length];
      return new TextDecoder().decode(bytes);
    } catch (err) {
      console.warn("failed to decode management key storage", err);
      return "";
    }
  }

  function authState() {
    var raw = localStorage.getItem(AUTH_KEY);
    if (!raw) return {};
    try {
      var decoded = decodeSecure(raw);
      var parsed = JSON.parse(decoded);
      return parsed && parsed.state ? parsed.state : {};
    } catch (err) {
      return {};
    }
  }

  function apiBase() {
    var state = authState();
    return (state.apiBase || window.location.origin).replace(/\/+$/, "");
  }

  function managementKey() {
    return authState().managementKey || "";
  }

  async function apiFetch(path, options) {
    var key = managementKey();
    if (!key) throw new Error("未读取到管理密钥，请重新登录管理面板后再试。");
    var headers = Object.assign({"Content-Type": "application/json"}, options && options.headers || {});
    headers.Authorization = "Bearer " + key;
    headers["X-Management-Key"] = key;
    var resp = await fetch(apiBase() + "/v0/management" + path, Object.assign({}, options || {}, {headers: headers}));
    if (!resp.ok) {
      var text = await resp.text();
      try {
        var json = JSON.parse(text);
        throw new Error(json.error || text || ("HTTP " + resp.status));
      } catch (err) {
        if (err && err.message && err.message !== text) throw err;
        throw new Error(text || ("HTTP " + resp.status));
      }
    }
    return resp.json();
  }

  async function apiDownload(path, options) {
    var key = managementKey();
    if (!key) throw new Error("未读取到管理密钥，请重新登录管理面板后再试。");
    var headers = Object.assign({"Content-Type": "application/json"}, options && options.headers || {});
    headers.Authorization = "Bearer " + key;
    headers["X-Management-Key"] = key;
    var resp = await fetch(apiBase() + "/v0/management" + path, Object.assign({}, options || {}, {headers: headers}));
    if (!resp.ok) {
      var text = await resp.text();
      try {
        var json = JSON.parse(text);
        throw new Error(json.error || text || ("HTTP " + resp.status));
      } catch (err) {
        if (err && err.message && err.message !== text) throw err;
        throw new Error(text || ("HTTP " + resp.status));
      }
    }
    var disposition = resp.headers.get("Content-Disposition") || "";
    var match = disposition.match(/filename="?([^";]+)"?/i);
    return {blob: await resp.blob(), filename: match ? match[1] : "codex-cards.txt"};
  }

  function saveBlob(blob, filename) {
    var url = URL.createObjectURL(blob);
    var link = document.createElement("a");
    link.href = url;
    link.download = filename || "codex-cards.txt";
    document.body.appendChild(link);
    link.click();
    link.remove();
    setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
  }

  function rememberCodexPage(active) {
    codexPageActive = active;
    try {
      if (active) window.sessionStorage.setItem(ACTIVE_KEY, "1");
      else window.sessionStorage.removeItem(ACTIVE_KEY);
    } catch (err) {
      // Ignore browsers that block sessionStorage.
    }
  }

  function isCodexPageActive() {
    if (window.location.hash === PAGE_HASH || codexPageActive) return true;
    try {
      return window.sessionStorage.getItem(ACTIVE_KEY) === "1";
    } catch (err) {
      return false;
    }
  }

  function removeCodexPage() {
    document.body.classList.remove("codex-card-admin-active");
    document.querySelectorAll(".codex-card-admin-page").forEach(function (node) {
      node.remove();
    });
  }

  function ensureNav() {
    var navSection = document.querySelector(".sidebar .nav-section");
    if (!navSection || navSection.querySelector('[data-codex-card-nav="true"]')) return;
    var item = document.createElement("a");
    item.className = "nav-item";
    item.href = PAGE_HASH;
    item.setAttribute("data-codex-card-nav", "true");
    item.innerHTML = '<span class="nav-icon">' + iconSVG() + '</span><span class="nav-label">卡密管理</span>';
    item.addEventListener("click", function (event) {
      event.preventDefault();
      rememberCodexPage(true);
      renderIfNeeded();
      setTimeout(renderIfNeeded, 80);
      setTimeout(renderIfNeeded, 260);
    });
    var authLink = Array.from(navSection.querySelectorAll("a")).find(function (node) {
      return node.getAttribute("href") === "#/auth-files";
    });
    if (authLink && authLink.nextSibling) {
      navSection.insertBefore(item, authLink.nextSibling);
    } else {
      navSection.appendChild(item);
    }
  }

  function setActiveNav(active) {
    document.querySelectorAll(".sidebar .nav-item").forEach(function (node) {
      if (node.getAttribute("data-codex-card-nav") === "true") {
        node.classList.toggle("active", active);
        if (active) node.setAttribute("aria-current", "page");
        else node.removeAttribute("aria-current");
      } else if (active) {
        node.classList.remove("active");
        node.removeAttribute("aria-current");
      }
    });
  }

  function escapeHTML(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function formatDate(value) {
    if (!value) return "-";
    var d = new Date(value);
    if (Number.isNaN(d.getTime())) return String(value);
    return d.toLocaleString();
  }

  function pageShell() {
    return ` + "`" + `
<div class="codex-card-admin-page">
  <section class="codex-card-admin-page-header">
    <h1 class="codex-card-admin-title">卡密管理</h1>
    <p class="codex-card-admin-desc">在管理端生成或导入卡密；用户只需要在公开提取页输入卡密，即可随机领取一个通过验活的 Codex 认证 JSON 文件。</p>
  </section>
  <section class="codex-card-admin-stats" id="codexCardStats">
    <div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">-</div><div class="codex-card-admin-stat-label">总卡密</div></div>
    <div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">-</div><div class="codex-card-admin-stat-label">未使用</div></div>
    <div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">-</div><div class="codex-card-admin-stat-label">已提取</div></div>
    <div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">-</div><div class="codex-card-admin-stat-label">已禁用</div></div>
  </section>
  <div class="codex-card-admin-grid">
    <section class="codex-card-admin-card">
      <h2>系统生成卡密</h2>
      <p class="codex-card-admin-muted">生成的卡密会保存到认证目录下的卡密库，状态默认为未使用。</p>
      <label class="codex-card-admin-label" for="codexCardGenerateCount">生成数量</label>
      <div class="codex-card-admin-row">
        <input class="codex-card-admin-input" id="codexCardGenerateCount" type="number" min="1" step="1" value="1">
        <button class="codex-card-admin-button" id="codexCardGenerateButton">生成卡密</button>
      </div>
      <div class="codex-card-admin-status" id="codexCardGenerateStatus"></div>
      <pre class="codex-card-admin-output" id="codexCardGenerateOutput">等待生成...</pre>
    </section>
    <section class="codex-card-admin-card">
      <h2>外部导入卡密</h2>
      <p class="codex-card-admin-muted">一行一个卡密；重复卡密不会覆盖已有兑换状态。</p>
      <label class="codex-card-admin-label" for="codexCardImportCodes">待导入卡密</label>
      <textarea class="codex-card-admin-textarea" id="codexCardImportCodes" placeholder="EXTERNAL-CARD-001&#10;EXTERNAL-CARD-002"></textarea>
      <div class="codex-card-admin-actions">
        <button class="codex-card-admin-button" id="codexCardImportButton">导入卡密</button>
        <a class="codex-card-admin-button secondary" href="/codex-extract.html" target="_blank" rel="noopener">打开用户提取页</a>
      </div>
      <div class="codex-card-admin-status" id="codexCardImportStatus"></div>
    </section>
    <section class="codex-card-admin-card wide">
      <div class="codex-card-admin-list-head">
        <div class="codex-card-admin-list-head-text">
          <h2>卡密列表</h2>
          <p class="codex-card-admin-muted">显示最近生成/导入的卡密及兑换状态，已兑换卡密会记录对应的 Codex 认证文件。</p>
        </div>
      </div>
      <div class="codex-card-admin-bulkbar">
        <div class="codex-card-admin-search">
          <input class="codex-card-admin-input" id="codexCardSearchInput" type="search" placeholder="搜索卡密、状态、来源或兑换文件">
        </div>
        <span class="codex-card-admin-selection" id="codexCardSelectionStatus">已选择 0 个</span>
        <div class="codex-card-admin-bulk-actions">
          <button class="codex-card-admin-button secondary" id="codexCardRefreshButton">刷新列表</button>
          <button class="codex-card-admin-button secondary" id="codexCardExportSelectedButton" disabled>导出选中</button>
          <button class="codex-card-admin-button secondary" id="codexCardExportAllButton">导出全部</button>
          <button class="codex-card-admin-button danger" id="codexCardDeleteSelectedButton" disabled>删除选中</button>
        </div>
      </div>
      <div class="codex-card-admin-status" id="codexCardListStatus"></div>
      <div class="codex-card-admin-table-wrap" id="codexCardTableWrap">
        <div class="codex-card-admin-empty">正在加载卡密...</div>
      </div>
    </section>
  </div>
</div>` + "`" + `;
  }

  function updateStatus(id, message, type) {
    var el = document.getElementById(id);
    if (!el) return;
    el.textContent = message || "";
    el.className = "codex-card-admin-status" + (type ? " " + type : "");
  }

  function renderStats(summary) {
    var root = document.getElementById("codexCardStats");
    if (!root) return;
    var items = [
      ["total", "总卡密"],
      ["unused", "未使用"],
      ["redeemed", "已提取"],
      ["disabled", "已禁用"]
    ];
    root.innerHTML = items.map(function (item) {
      return '<div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">' + escapeHTML(summary && summary[item[0]] || 0) + '</div><div class="codex-card-admin-stat-label">' + item[1] + '</div></div>';
    }).join("");
  }

  function cardMatchesSearch(card, query) {
    if (!query) return true;
    if (!card) return false;
    var haystack = [
      card.code,
      card.status,
      card.source,
      card.created_at,
      card.redeemed_file,
      card.redeemed_auth_id,
      card.note
    ].join(" ").toLowerCase();
    return haystack.indexOf(query) >= 0;
  }

  function filteredCards() {
    var input = document.getElementById("codexCardSearchInput");
    var query = input ? String(input.value || "").trim().toLowerCase() : "";
    return (allCards || []).filter(function (card) {
      return cardMatchesSearch(card, query);
    });
  }

  function applyCardSearch() {
    renderTable(filteredCards());
  }

  function renderTable(cards) {
    var wrap = document.getElementById("codexCardTableWrap");
    if (!wrap) return;
    currentCards = Array.isArray(cards) ? cards : [];
    if (!cards || cards.length === 0) {
      var searchInput = document.getElementById("codexCardSearchInput");
      var message = searchInput && searchInput.value.trim() ? "没有匹配的卡密，请换个关键词。" : "还没有卡密，先生成或导入一批。";
      wrap.innerHTML = '<div class="codex-card-admin-empty">' + escapeHTML(message) + '</div>';
      updateSelectionControls();
      return;
    }
    wrap.innerHTML = '<table class="codex-card-admin-table"><thead><tr><th class="select"><input class="codex-card-admin-checkbox" id="codexCardSelectAllTable" type="checkbox" aria-label="全选卡密"></th><th>卡密</th><th>状态</th><th>来源</th><th>创建时间</th><th>兑换文件</th></tr></thead><tbody>' + cards.map(function (card) {
      var status = card.status || "";
      var file = card.redeemed_file ? '<span class="codex-card-admin-code">' + escapeHTML(card.redeemed_file) + '</span>' : "-";
      return '<tr><td class="select"><input class="codex-card-admin-checkbox codex-card-row-checkbox" type="checkbox" value="' + escapeHTML(card.code) + '" aria-label="选择卡密 ' + escapeHTML(card.code) + '"></td><td class="codex-card-admin-code">' + escapeHTML(card.code) + '</td><td><span class="codex-card-admin-pill ' + escapeHTML(status) + '">' + escapeHTML(status) + '</span></td><td>' + escapeHTML(card.source || "-") + '</td><td>' + escapeHTML(formatDate(card.created_at)) + '</td><td>' + file + '</td></tr>';
    }).join("") + '</tbody></table>';
    bindTableSelection();
    updateSelectionControls();
  }

  function selectedCardCodes() {
    return Array.from(document.querySelectorAll(".codex-card-row-checkbox:checked")).map(function (node) {
      return node.value;
    }).filter(Boolean);
  }

  function setAllCardCheckboxes(checked) {
    document.querySelectorAll(".codex-card-row-checkbox").forEach(function (node) {
      node.checked = checked;
    });
    updateSelectionControls();
  }

  function updateSelectionControls() {
    var total = document.querySelectorAll(".codex-card-row-checkbox").length;
    var selected = selectedCardCodes().length;
    var status = document.getElementById("codexCardSelectionStatus");
    if (status) status.textContent = "已选择 " + selected + " 个";
    ["codexCardExportSelectedButton", "codexCardDeleteSelectedButton"].forEach(function (id) {
      var button = document.getElementById(id);
      if (button) button.disabled = selected === 0;
    });
    ["codexCardSelectAllTable"].forEach(function (id) {
      var checkbox = document.getElementById(id);
      if (!checkbox) return;
      checkbox.checked = total > 0 && selected === total;
      checkbox.indeterminate = selected > 0 && selected < total;
      checkbox.disabled = total === 0;
    });
    var exportAll = document.getElementById("codexCardExportAllButton");
    if (exportAll) exportAll.disabled = allCards.length === 0;
  }

  function bindTableSelection() {
    var tableSelectAll = document.getElementById("codexCardSelectAllTable");
    if (tableSelectAll) {
      tableSelectAll.addEventListener("change", function () {
        setAllCardCheckboxes(tableSelectAll.checked);
      });
    }
    document.querySelectorAll(".codex-card-row-checkbox").forEach(function (node) {
      node.addEventListener("change", updateSelectionControls);
    });
  }

  async function loadCards() {
    updateStatus("codexCardListStatus", "正在加载卡密列表...", "");
    try {
      var data = await apiFetch("/codex-cards", {method: "GET", headers: {}});
      allCards = Array.isArray(data.cards) ? data.cards : [];
      renderStats(data.summary || {});
      renderTable(filteredCards());
      updateStatus("codexCardListStatus", "卡密列表已刷新。", "ok");
    } catch (err) {
      updateStatus("codexCardListStatus", err.message || String(err), "error");
    }
  }

  async function exportCards(all) {
    var codes = selectedCardCodes();
    if (!all && codes.length === 0) {
      updateStatus("codexCardListStatus", "请先勾选要导出的卡密。", "error");
      return;
    }
    updateStatus("codexCardListStatus", all ? "正在导出全部卡密..." : "正在导出选中卡密...", "");
    var data = await apiDownload("/codex-cards/export", {method: "POST", body: JSON.stringify(all ? {all: true} : {items: codes})});
    saveBlob(data.blob, data.filename);
    updateStatus("codexCardListStatus", all ? "全部卡密已导出。" : "已导出 " + codes.length + " 个选中卡密。", "ok");
  }

  async function deleteSelectedCards() {
    var codes = selectedCardCodes();
    if (codes.length === 0) {
      updateStatus("codexCardListStatus", "请先勾选要删除的卡密。", "error");
      return;
    }
    if (!window.confirm("确定删除选中的 " + codes.length + " 个卡密吗？删除后这些卡密将不能再用于提取。")) {
      return;
    }
    updateStatus("codexCardListStatus", "正在删除选中卡密...", "");
    var data = await apiFetch("/codex-cards/delete", {method: "POST", body: JSON.stringify({items: codes})});
    updateStatus("codexCardListStatus", "已删除 " + data.deleted + " 个卡密。", "ok");
    await loadCards();
  }

  function bindPage() {
    var generateButton = document.getElementById("codexCardGenerateButton");
    if (generateButton) {
      generateButton.addEventListener("click", async function () {
        generateButton.disabled = true;
        updateStatus("codexCardGenerateStatus", "正在生成卡密...", "");
        try {
          var count = Number(document.getElementById("codexCardGenerateCount").value || "1");
          var data = await apiFetch("/codex-cards/generate", {method: "POST", body: JSON.stringify({count: count})});
          var codes = data.codes || [];
          document.getElementById("codexCardGenerateOutput").textContent = codes.join("\n") || JSON.stringify(data, null, 2);
          updateStatus("codexCardGenerateStatus", "已生成 " + codes.length + " 个卡密。", "ok");
          await loadCards();
        } catch (err) {
          updateStatus("codexCardGenerateStatus", err.message || String(err), "error");
        } finally {
          generateButton.disabled = false;
        }
      });
    }
    var importButton = document.getElementById("codexCardImportButton");
    if (importButton) {
      importButton.addEventListener("click", async function () {
        importButton.disabled = true;
        updateStatus("codexCardImportStatus", "正在导入卡密...", "");
        try {
          var codes = document.getElementById("codexCardImportCodes").value || "";
          var data = await apiFetch("/codex-cards/import", {method: "POST", body: JSON.stringify({codes: codes})});
          updateStatus("codexCardImportStatus", "导入 " + data.imported + " 个，重复 " + ((data.duplicates || []).length) + " 个，非法 " + ((data.invalid || []).length) + " 个。", "ok");
          await loadCards();
        } catch (err) {
          updateStatus("codexCardImportStatus", err.message || String(err), "error");
        } finally {
          importButton.disabled = false;
        }
      });
    }
    var refreshButton = document.getElementById("codexCardRefreshButton");
    if (refreshButton) refreshButton.addEventListener("click", loadCards);
    var searchInput = document.getElementById("codexCardSearchInput");
    if (searchInput) {
      searchInput.addEventListener("input", applyCardSearch);
    }
    var exportSelectedButton = document.getElementById("codexCardExportSelectedButton");
    if (exportSelectedButton) {
      exportSelectedButton.addEventListener("click", async function () {
        exportSelectedButton.disabled = true;
        try {
          await exportCards(false);
        } catch (err) {
          updateStatus("codexCardListStatus", err.message || String(err), "error");
        } finally {
          updateSelectionControls();
        }
      });
    }
    var exportAllButton = document.getElementById("codexCardExportAllButton");
    if (exportAllButton) {
      exportAllButton.addEventListener("click", async function () {
        exportAllButton.disabled = true;
        try {
          await exportCards(true);
        } catch (err) {
          updateStatus("codexCardListStatus", err.message || String(err), "error");
        } finally {
          updateSelectionControls();
        }
      });
    }
    var deleteButton = document.getElementById("codexCardDeleteSelectedButton");
    if (deleteButton) {
      deleteButton.addEventListener("click", async function () {
        deleteButton.disabled = true;
        try {
          await deleteSelectedCards();
        } catch (err) {
          updateStatus("codexCardListStatus", err.message || String(err), "error");
        } finally {
          updateSelectionControls();
        }
      });
    }
  }

  function renderIfNeeded() {
    ensureStyles();
    ensureNav();
    var isPage = isCodexPageActive();
    setActiveNav(isPage);
    if (!isPage) {
      removeCodexPage();
      return;
    }
    var main = document.querySelector(".main-content");
    if (!main) return;
    document.body.classList.add("codex-card-admin-active");
    if (main.querySelector(".codex-card-admin-page")) return;
    var token = ++lastRenderToken;
    main.insertAdjacentHTML("beforeend", pageShell());
    bindPage();
    loadCards();
    setTimeout(function () {
      if (token === lastRenderToken) setActiveNav(true);
    }, 80);
  }

  function boot() {
    ensureStyles();
    ensureNav();
    renderIfNeeded();
    if (!observerStarted) {
      observerStarted = true;
      var observer = new MutationObserver(function () {
        ensureNav();
        if (isCodexPageActive()) renderIfNeeded();
        else {
          setActiveNav(false);
          removeCodexPage();
        }
      });
      observer.observe(document.body, {childList: true, subtree: true});
      document.addEventListener("click", function (event) {
        var navItem = event.target && event.target.closest ? event.target.closest(".sidebar .nav-item") : null;
        if (navItem && navItem.getAttribute("data-codex-card-nav") !== "true") {
          rememberCodexPage(false);
          removeCodexPage();
        }
      }, true);
      window.addEventListener("hashchange", function () {
        if (window.location.hash === PAGE_HASH) {
          rememberCodexPage(true);
        } else if (window.location.hash && window.location.hash !== "#/") {
          rememberCodexPage(false);
          removeCodexPage();
        }
        setTimeout(renderIfNeeded, 80);
      });
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () {
      setTimeout(boot, 100);
    });
  } else {
    setTimeout(boot, 100);
  }
})();
`
