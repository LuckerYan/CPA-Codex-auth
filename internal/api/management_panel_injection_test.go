package api

import (
	"bytes"
	"testing"
)

func TestPatchQuotaManagementPanel(t *testing.T) {
	input := []byte(
		"var pb=25,mb=30," +
			"[c,l]=fb(380),[u,d]=(0,y.useState)(`paged`),[f,p]=(0,y.useState)(!1),m=(0,y.useMemo)" +
			",(0,y.useEffect)(()=>{S(g===`all`?Math.max(1,m.length):Math.min(c*3,pb))},[g,c,m.length,S]);" +
			"let t=g===`all`?`all`:`page`,r=g===`all`?m:x;r.length!==0&&O(r,t,E)" +
			"children:[(0,B.jsxs)(`div`,{className:sb.viewModeToggle,children:[" +
			"(0,B.jsx)(V,{variant:`secondary`,size:`sm`,className:`${sb.viewModeButton} ${g===`all`?sb.viewModeButtonActive:``}`,onClick:()=>{m.length>mb?p(!0):d(`all`)},children:i(`auth_files.view_mode_all`)})",
	)

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "var pb=25,mb=1e3,")
	assertContains(t, patched, "[q,z]=(0,y.useState)(``)")
	assertContains(t, patched, "g===`paged`&&(0,B.jsx)(`input`")
	assertContains(t, patched, "style:{width:160}")
	assertContains(t, patched, "value:q||String(_)")
	assertContains(t, patched, "title:i(`auth_files.page_size_label`)")
	assertContains(t, patched, "onClick:()=>d(`all`)")
	assertContains(t, patched, "O(t,`all`,E)")
	assertContains(t, patched, ";let qn=Math.min(c*3,pb)")
	assertNotContains(t, patched, "onClick:()=>{m.length>mb?p(!0):d(`all`)}")
	assertNotContains(t, patched, "let t=g===`all`?`all`:`page`")
}

func TestPatchQuotaManagementPanelThrottlesRefreshAll(t *testing.T) {
	input := []byte("let i=await Promise.all(n.map(async n=>{try{let r=await e.fetchQuota(n,t);return{name:n.name,status:`success`,data:r}}catch(e){let r=e instanceof Error?e.message:t(`common.unknown_error`),i=Ry(e);return{name:n.name,status:`error`,error:r,errorStatus:i}}}));if(c!==a.current)return;r(n=>{let r={...n};return i.forEach(n=>{n.status===`success`?r[n.name]=e.buildSuccessState(n.data):r[n.name]=e.buildErrorState(n.error||t(`common.unknown_error`),n.errorStatus)}),r})")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "window.__CPA_QUOTA_REFRESH_CONCURRENCY")
	assertContains(t, patched, "Math.max(1,Math.min(10")
	assertContains(t, patched, "||10")
	assertContains(t, patched, "Array.from({length:Math.min(d,n.length)},()=>f())")
	assertContains(t, patched, "c===a.current&&r(t=>({...t,[o.name]:e.buildSuccessState(n)}))")
	assertNotContains(t, patched, "Promise.all(n.map(async n=>")
}

func TestPatchQuotaManagementPanelDispatchesAuthFilesUpdateEvent(t *testing.T) {
	input := []byte("finally{c===a.current&&(s(!1),i.current=!1)}}")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "cli-proxy-auth-files-updated")
	assertContains(t, patched, "source:`quota-refresh`")
	assertContains(t, patched, "type:e.type")
	assertContains(t, patched, "scope:o")
	assertNotContains(t, patched, "finally{c===a.current&&(s(!1),i.current=!1)}}")
}

func TestPatchAuthFilesPageRefreshesAfterQuotaUpdateEvent(t *testing.T) {
	input := []byte("a_(ot),(0,y.useEffect)(()=>{a&&(ce(),Oe(),ke())},[a,ce,Oe,ke])")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "window.addEventListener(`cli-proxy-auth-files-updated`,e)")
	assertContains(t, patched, "window.removeEventListener(`cli-proxy-auth-files-updated`,e)")
	assertContains(t, patched, "window.location.hash===`#/auth-files`&&ot().catch(()=>{})")
	assertContains(t, patched, "(0,y.useEffect)(()=>{a&&(ce(),Oe(),ke())},[a,ce,Oe,ke])")
}

func TestPatchAuthFilesUploadResponseIncludesDuplicateCount(t *testing.T) {
	input := []byte("Bh=e=>Array.isArray(e)?zh(e.map(e=>String(e??``))):[],Vh=e=>Array.isArray(e)?e.reduce((e,t)=>{if(!t||typeof t!=`object`)return e;let n=t,r=String(n.name??``).trim(),i=typeof n.error==`string`?n.error.trim():typeof n.message==`string`?n.message.trim():``;return!r&&!i||e.push({name:r,error:i||`Unknown error`}),e},[]):[],Hh=(e,t)=>{let n=new Set(t.map(e=>e.name.trim()).filter(Boolean));return n.size===0?[...e]:e.filter(e=>!n.has(e))},Uh=(e,t)=>{let n=Vh(e?.failed),r=Bh(e?.files),i=typeof e?.uploaded==`number`?e.uploaded:r.length>0?r.length:+(t.length===1&&n.length===0),a=r;if(a.length===0&&i>0)if(n.length===0&&i===t.length)a=[...t];else{let e=Hh(t,n);e.length===i&&(a=e)}return{status:typeof e?.status==`string`?e.status:n.length>0?`partial`:`ok`,uploaded:i,files:a,failed:n}},Wh=")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "duplicates:i")
	assertContains(t, patched, "i.length>0&&a===0?`duplicate`")
	assertNotContains(t, patched, "return{status:typeof e?.status==`string`?e.status:n.length>0?`partial`:`ok`,uploaded:i,files:a,failed:n}},Wh=")
}

func TestPatchAuthFilesUploadToastShowsDuplicateCount(t *testing.T) {
	input := []byte("let n=await ag.uploadFiles(a),r=n.uploaded;if(r>0){let i=a.length>1?` (${r}/${a.length})`:``;t(`${e(`auth_files.upload_success`)}${i}`,n.failed.length?`warning`:`success`),await A()}if(n.failed.length>0){let r=n.failed.map(e=>`${e.name}: ${e.error}`).join(`; `);t(`${e(`notification.upload_failed`)}: ${r}`,`error`)}")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "Array.isArray(n.duplicates)?n.duplicates.length:0")
	assertContains(t, patched, "成功 ${r} / 重复 ${u} / 失败 ${f} / 总计 ${a.length}")
	assertContains(t, patched, "notification.upload_failed")
	assertNotContains(t, patched, "if(n.failed.length>0){let r=n.failed.map(e=>`${e.name}: ${e.error}`).join(`; `);t(`${e(`notification.upload_failed`)}: ${r}`,`error`)}")
}

func TestPatchAuthFilesDisplayOptionsDropdown(t *testing.T) {
	input := []byte("(0,B.jsxs)(`div`,{className:`${G.filterItem} ${G.filterToggleItem}`,children:[(0,B.jsx)(`label`,{children:e(`auth_files.display_options_label`)}),(0,B.jsxs)(`div`,{className:G.filterToggleGroup,children:[(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:l,onChange:e=>{u(e),v(1)},ariaLabel:e(`auth_files.problem_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.problem_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:d,onChange:e=>{f(e),v(1)},ariaLabel:e(`auth_files.disabled_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.disabled_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:p,onChange:e=>m(e),ariaLabel:e(`auth_files.compact_mode_label`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.compact_mode_label`)})})})]})]})")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "auth-files-display-options-menu")
	assertContains(t, patched, "auth-files-display-options-trigger")
	assertContains(t, patched, "auth-files-display-options-list")
	assertContains(t, patched, "children:(l?1:0)+(d?1:0)+(extractedOnly?1:0)+(unextractedOnly?1:0)+(p?1:0)")
	assertContains(t, patched, "仅显示未提取凭证")
	assertContains(t, patched, "仅显示已提取凭证")
	assertContains(t, patched, "e&&setExtractedOnly(!1)")
	assertContains(t, patched, "e&&setUnextractedOnly(!1)")
	assertNotContains(t, patched, "className:G.filterToggleGroup,children:[")
}

func TestPatchAuthFilesExtractionFilters(t *testing.T) {
	input := []byte(
		"[s,c]=(0,y.useState)(`all`),[l,u]=(0,y.useState)(!1),[d,f]=(0,y.useState)(!1),[p,m]=(0,y.useState)(!1),[h,g]=(0,y.useState)(``)," +
			"typeof t.problemOnly==`boolean`&&u(t.problemOnly),typeof t.disabledOnly==`boolean`&&f(t.disabledOnly),typeof e!=`boolean`&&typeof t.compactMode==`boolean`&&m(t.compactMode)," +
			"zx({filter:s,problemOnly:l,disabledOnly:d,compactMode:p,search:h,page:_,pageSize:tt,regularPageSize:b.regular,compactPageSize:b.compact,sortMode:D}),Vx(p))},[p,d,s,_,tt,b,l,h,D,j])" +
			"let st=(0,y.useMemo)(()=>{let e=new Set([`all`]);return I.forEach(t=>{t.type&&e.add(t.type)}),Array.from(e)},[I]),ct=(0,y.useMemo)(()=>I.filter(e=>!(l&&!Vv(e)||d&&e.disabled!==!0)),[d,I,l]),lt=",
	)

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "[extractedOnly,setExtractedOnly]=(0,y.useState)(!1)")
	assertContains(t, patched, "[unextractedOnly,setUnextractedOnly]=(0,y.useState)(!1)")
	assertContains(t, patched, "typeof t.extractedOnly==`boolean`&&setExtractedOnly(t.extractedOnly)")
	assertContains(t, patched, "extractedOnly:extractedOnly")
	assertContains(t, patched, "unextractedOnly:unextractedOnly")
	assertContains(t, patched, "codexExtractedFilterMatch")
	assertContains(t, patched, "codexUnextractedFilterMatch")
	assertContains(t, patched, "extractedOnly&&!codexExtractedFilterMatch(e)")
	assertContains(t, patched, "unextractedOnly&&!codexUnextractedFilterMatch(e)")
}

func TestPatchAuthFilesCardQuotaRefreshButton(t *testing.T) {
	input := []byte(
		"function ex(e){let{t}=qo(),{file:n,compact:r,selected:i,resolvedTheme:a,disableControls:o,deleting:s,statusUpdating:c,quotaFilterType:l,statusBarCache:u,onShowModels:d,onDownload:f,onOpenPrefixProxyEditor:p,onDelete:m,onToggleStatus:h,onToggleSelect:g}=e,_=" +
			"!y&&(0,B.jsxs)(`div`,{className:G.statusToggle,children:[(0,B.jsx)(`span`,{className:G.statusToggleLabel,children:t(`auth_files.status_toggle_label`)}),(0,B.jsx)(Sg,{ariaLabel:t(`auth_files.status_toggle_label`),checked:!n.disabled,disabled:o||c[n.name]===!0,onChange:e=>h(n,e)})]})",
	)

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "codexQuotaForCard=np(e=>e.codexQuota[n.name])")
	assertContains(t, patched, "setCodexQuotaForCard=np(e=>e.setCodexQuota)")
	assertContains(t, patched, "refreshCodexQuotaForCard")
	assertContains(t, patched, "Xb(`codex`)")
	assertContains(t, patched, "auth-file-card-quota-refresh-button")
	assertContains(t, patched, "title:`刷新额度`")
	assertNotContains(t, patched, "children:`刷新`")
}

func TestPatchAuthFilesSortSelectChevron(t *testing.T) {
	input := []byte("(0,B.jsx)(`span`,{className:Is.triggerIcon,\"aria-hidden\":`true`,children:(0,B.jsx)(fs,{size:14})})")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "className:Is.triggerIcon")
	assertContains(t, patched, "children:(0,B.jsx)(fs,{size:14})")
	assertNotContains(t, patched, "String(i??``).includes(`AuthFilesPage-module__sortSelect`)?null")
}

func TestPatchSelectDropdownAlwaysDown(t *testing.T) {
	input := []byte("Hs=e=>{let t=e.getBoundingClientRect(),n=window.innerWidth,r=window.innerHeight,i=Math.min(t.width,Math.max(0,n-Ls*2)),a=Vs(t.left,Ls,Math.max(Ls,n-i-Ls)),o=r-t.bottom-Ls-Rs,s=t.top-Ls-Rs,c=o>=zs||o>=s?`down`:`up`,l=Math.max(0,Math.min(zs,c===`down`?o:s));return c===`down`?{position:`fixed`,top:t.bottom+Rs,left:a,width:i,maxHeight:l,zIndex:Bs}:{position:`fixed`,bottom:r-t.top+Rs,left:a,width:i,maxHeight:l,zIndex:Bs}}")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "s=Math.max(0,Math.min(zs,o))")
	assertContains(t, patched, "return{position:`fixed`,top:t.bottom+Rs,left:a,width:i,maxHeight:s,zIndex:Bs}")
	assertNotContains(t, patched, "bottom:r-t.top+Rs")
	assertNotContains(t, patched, "c=o>=zs||o>=s?`down`:`up`")
}

func TestCodexCardManagementPanelExtractsKeyFromKeycodeLinks(t *testing.T) {
	script := []byte(codexCardManagementPanelScript)

	assertContains(t, script, "user@example.com---https://mail.lucker.cc.cd/keycode")
	assertContains(t, script, "mail.lucker.cc.cd/keycode?email")
	assertContains(t, script, "function extractCardCodeInput")
	assertContains(t, script, "function cardCodeInputCandidates")
	assertContains(t, script, "trimmed.indexOf(\"---\")")
	assertContains(t, script, "searchParams.get(\"key\")")
	assertContains(t, script, "function extractCardCodeInputs")
	assertContains(t, script, "JSON.stringify({items: codes})")
}

func TestCodexCardManagementPanelRemovesInlineHelpAndCopiesGeneratedCards(t *testing.T) {
	script := []byte(codexCardManagementPanelScript)

	assertNotContains(t, script, "生成的卡密会保存到认证目录下的卡密库")
	assertNotContains(t, script, "一行一个卡密或邮箱---keycode 链接；导入时会自动提取链接中的 key 参数")
	assertContains(t, script, "async function copyTextToClipboard(text)")
	assertContains(t, script, "navigator.clipboard.writeText(value)")
	assertContains(t, script, "document.execCommand(\"copy\")")
	assertContains(t, script, "var outputText = codes.join(\"\\n\") || JSON.stringify(data, null, 2);")
	assertContains(t, script, "await copyTextToClipboard(outputText)")
	assertContains(t, script, "已复制到剪贴板")
}

func TestCodexCardManagementPanelIncludesAuthFilesFilterStyles(t *testing.T) {
	script := []byte(codexCardManagementPanelScript)

	assertContains(t, script, ".auth-files-display-options-menu")
	assertContains(t, script, ".auth-files-display-options-list")
	assertContains(t, script, ".auth-file-card-quota-refresh-button")
	assertContains(t, script, "ToggleSwitch-module__root")
	assertContains(t, script, "grid-template-columns:minmax(220px,380px)")
	assertContains(t, script, "left:0;right:auto")
	assertContains(t, script, "width:100%;min-width:0")
	assertContains(t, script, "::-webkit-details-marker{display:none!important}")
}

func TestAuthFileCodexStatsCountsUnextractedOnlyForNormalFiles(t *testing.T) {
	script := []byte(authFileCodexStatsScript)

	assertContains(t, script, "var banned = isBannedFile(file);")
	assertContains(t, script, "if (banned) {")
	assertContains(t, script, "stats.banned += 1;")
	assertContains(t, script, "} else {\n        stats.normal += 1;")
	assertContains(t, script, "if (isExtractedFile(file)) stats.extracted += 1;")
	assertContains(t, script, "else if (!banned) stats.unextracted += 1;")
	assertNotContains(t, script, "stats.banned += 1;\n        return;")
	assertContains(t, script, "未提取=状态正常且尚未分配给用户")
	assertContains(t, script, "已提取=已分配给用户")
}

func TestAuthFileCodexStatsRefreshesAfterQuotaUpdateEvent(t *testing.T) {
	script := []byte(authFileCodexStatsScript)

	assertContains(t, script, "window.addEventListener(\"cli-proxy-auth-files-updated\"")
	assertContains(t, script, "setTimeout(function () { refreshStats(true); }, 150);")
}

func TestAuthFileCodexStatsForcesInitialReloadAfterNavigation(t *testing.T) {
	script := []byte(authFileCodexStatsScript)

	assertContains(t, script, "var pendingStatsRefresh = false;")
	assertContains(t, script, "lastFetchAt = 0;")
	assertContains(t, script, "panel.dataset.codexStatsLoaded = \"0\";")
	assertContains(t, script, "var needsInitialLoad = panel.dataset.codexStatsLoaded !== \"1\";")
	assertContains(t, script, "if (!force && !needsInitialLoad && now - lastFetchAt < 4000) return;")
	assertContains(t, script, "currentPanel.dataset.codexStatsLoaded = \"1\";")
	assertContains(t, script, "window.addEventListener(\"hashchange\", function () {\n      lastFetchAt = 0;")
}

func TestAuthFileCodexStatsRerunsPendingRefresh(t *testing.T) {
	script := []byte(authFileCodexStatsScript)

	assertContains(t, script, "if (fetching) {\n      pendingStatsRefresh = true;")
	assertContains(t, script, "pendingStatsRefresh = false;\n    lastFetchAt = now;")
	assertContains(t, script, "if (pendingStatsRefresh && window.location.hash === AUTH_FILES_HASH)")
	assertContains(t, script, "setTimeout(function () { refreshStats(true); }, 50);")
}

func TestAuthFilesDisplayOptionsDropdownClosesOnOutsideClick(t *testing.T) {
	script := []byte(authFileCodexStatsScript)

	assertContains(t, script, "function closeDisplayOptionsMenus(event)")
	assertContains(t, script, ".auth-files-display-options-menu[open]")
	assertContains(t, script, "if (target && menu.contains(target)) return;")
	assertContains(t, script, "menu.removeAttribute(\"open\")")
	assertContains(t, script, "document.addEventListener(\"pointerdown\", closeDisplayOptionsMenus, true)")
	assertContains(t, script, "event.key === \"Escape\"")
}

func assertContains(t *testing.T, data []byte, want string) {
	t.Helper()
	if !bytes.Contains(data, []byte(want)) {
		t.Fatalf("patched data does not contain %q", want)
	}
}

func assertNotContains(t *testing.T, data []byte, want string) {
	t.Helper()
	if bytes.Contains(data, []byte(want)) {
		t.Fatalf("patched data still contains %q", want)
	}
}
