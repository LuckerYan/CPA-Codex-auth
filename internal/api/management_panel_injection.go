package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"

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
	data = patchQuotaManagementPanel(data)
	data = injectManagementPanelScript(data, "codex-card-management-injection", codexCardManagementPanelScript)
	data = injectManagementPanelScript(data, "auth-file-codex-stats-injection", authFileCodexStatsScript)
	return data
}

func injectManagementPanelScript(data []byte, id string, body string) []byte {
	if len(data) == 0 || strings.TrimSpace(id) == "" || body == "" {
		return data
	}
	if bytes.Contains(data, []byte(id)) {
		return data
	}
	marker := []byte("</body>")
	script := []byte(fmt.Sprintf("\n<script id=\"%s\">\n%s\n</script>\n", id, body))
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

func patchQuotaManagementPanel(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// The management panel is served from a single-file release asset, so keep
	// quota-page compatibility fixes in the HTML response until the upstream
	// asset ships the same behavior.
	replacements := []struct {
		old string
		new string
	}{
		{
			old: "var pb=25,mb=30,",
			new: "var pb=25,mb=1e3,",
		},
		{
			old: "(0,B.jsx)(`span`,{className:Is.triggerIcon,\"aria-hidden\":`true`,children:(0,B.jsx)(fs,{size:14})})",
			new: "(0,B.jsx)(`span`,{className:Is.triggerIcon,\"aria-hidden\":`true`,children:(0,B.jsx)(fs,{size:14})})",
		},
		{
			old: "Hs=e=>{let t=e.getBoundingClientRect(),n=window.innerWidth,r=window.innerHeight,i=Math.min(t.width,Math.max(0,n-Ls*2)),a=Vs(t.left,Ls,Math.max(Ls,n-i-Ls)),o=r-t.bottom-Ls-Rs,s=t.top-Ls-Rs,c=o>=zs||o>=s?`down`:`up`,l=Math.max(0,Math.min(zs,c===`down`?o:s));return c===`down`?{position:`fixed`,top:t.bottom+Rs,left:a,width:i,maxHeight:l,zIndex:Bs}:{position:`fixed`,bottom:r-t.top+Rs,left:a,width:i,maxHeight:l,zIndex:Bs}}",
			new: "Hs=e=>{let t=e.getBoundingClientRect(),n=window.innerWidth,r=window.innerHeight,i=Math.min(t.width,Math.max(0,n-Ls*2)),a=Vs(t.left,Ls,Math.max(Ls,n-i-Ls)),o=r-t.bottom-Ls-Rs,s=Math.max(0,Math.min(zs,o));return{position:`fixed`,top:t.bottom+Rs,left:a,width:i,maxHeight:s,zIndex:Bs}}",
		},
		{
			old: "[c,l]=fb(380),[u,d]=(0,y.useState)(`paged`),[f,p]=(0,y.useState)(!1),m=(0,y.useMemo)",
			new: "[c,l]=fb(380),[u,d]=(0,y.useState)(`paged`),[q,z]=(0,y.useState)(``),[f,p]=(0,y.useState)(!1),m=(0,y.useMemo)",
		},
		{
			old: ",(0,y.useEffect)(()=>{S(g===`all`?Math.max(1,m.length):Math.min(c*3,pb))},[g,c,m.length,S]);",
			new: ";let qn=Math.min(c*3,pb),zn=(()=>{let e=Number(q);return!Number.isFinite(e)||e<=0?null:Math.max(1,Math.min(Math.round(e),Math.max(m.length,pb)))})();(0,y.useEffect)(()=>{if(g===`all`){S(Math.max(1,m.length));return}S(zn??qn)},[g,m.length,zn,qn,S]);",
		},
		{
			old: "let t=g===`all`?`all`:`page`,r=g===`all`?m:x;r.length!==0&&O(r,t,E)",
			new: "let t=m;t.length!==0&&O(t,`all`,E)",
		},
		{
			old: "children:[(0,B.jsxs)(`div`,{className:sb.viewModeToggle,children:[",
			new: "children:[g===`paged`&&(0,B.jsx)(`input`,{className:sb.pageSizeSelect,style:{width:160},type:`number`,min:`1`,step:`1`,inputMode:`numeric`,value:q||String(_),title:i(`auth_files.page_size_label`),\"aria-label\":i(`auth_files.page_size_label`),onFocus:()=>d(`paged`),onChange:e=>{d(`paged`),z(e.target.value.replace(/[^0-9]/g,``))}}),(0,B.jsxs)(`div`,{className:sb.viewModeToggle,children:[",
		},
		{
			old: "let i=await Promise.all(n.map(async n=>{try{let r=await e.fetchQuota(n,t);return{name:n.name,status:`success`,data:r}}catch(e){let r=e instanceof Error?e.message:t(`common.unknown_error`),i=Ry(e);return{name:n.name,status:`error`,error:r,errorStatus:i}}}));if(c!==a.current)return;r(n=>{let r={...n};return i.forEach(n=>{n.status===`success`?r[n.name]=e.buildSuccessState(n.data):r[n.name]=e.buildErrorState(n.error||t(`common.unknown_error`),n.errorStatus)}),r})",
			new: "let u=0,d=Math.max(1,Math.min(10,Math.floor(Number(window.__CPA_QUOTA_REFRESH_CONCURRENCY)||10))),f=async()=>{for(;;){let i=u++;if(i>=n.length)return;let o=n[i];try{let n=await e.fetchQuota(o,t);c===a.current&&r(t=>({...t,[o.name]:e.buildSuccessState(n)}))}catch(n){let i=n instanceof Error?n.message:t(`common.unknown_error`),s=Ry(n);c===a.current&&r(t=>({...t,[o.name]:e.buildErrorState(i,s)}))}}};await Promise.all(Array.from({length:Math.min(d,n.length)},()=>f()))",
		},
		{
			old: "finally{c===a.current&&(s(!1),i.current=!1)}}",
			new: "finally{c===a.current&&(s(!1),i.current=!1,typeof window<`u`&&window.dispatchEvent(new CustomEvent(`cli-proxy-auth-files-updated`,{detail:{source:`quota-refresh`,type:e.type,scope:o}})))}}",
		},
		{
			old: "a_(ot),(0,y.useEffect)(()=>{a&&(ce(),Oe(),ke())},[a,ce,Oe,ke])",
			new: "a_(ot),(0,y.useEffect)(()=>{let e=()=>{window.location.hash===`#/auth-files`&&ot().catch(()=>{})};return window.addEventListener(`cli-proxy-auth-files-updated`,e),()=>window.removeEventListener(`cli-proxy-auth-files-updated`,e)},[ot]),(0,y.useEffect)(()=>{a&&(ce(),Oe(),ke())},[a,ce,Oe,ke])",
		},
		{
			old: "Vv=e=>Bv(e).length>0",
			new: "Vv=e=>Bv(e).length>0||String(e.account_status??e.accountStatus??e.status??``).trim().toLowerCase()===`banned`",
		},
		{
			old: "[s,c]=(0,y.useState)(`all`),[l,u]=(0,y.useState)(!1),[d,f]=(0,y.useState)(!1),[p,m]=(0,y.useState)(!1),[h,g]=(0,y.useState)(``),",
			new: "[s,c]=(0,y.useState)(`all`),[l,u]=(0,y.useState)(!1),[d,f]=(0,y.useState)(!1),[extractedOnly,setExtractedOnly]=(0,y.useState)(!1),[unextractedOnly,setUnextractedOnly]=(0,y.useState)(!1),[p,m]=(0,y.useState)(!1),[h,g]=(0,y.useState)(``),",
		},
		{
			old: "typeof t.problemOnly==`boolean`&&u(t.problemOnly),typeof t.disabledOnly==`boolean`&&f(t.disabledOnly),typeof e!=`boolean`&&typeof t.compactMode==`boolean`&&m(t.compactMode),",
			new: "typeof t.problemOnly==`boolean`&&u(t.problemOnly),typeof t.disabledOnly==`boolean`&&f(t.disabledOnly),typeof t.extractedOnly==`boolean`&&setExtractedOnly(t.extractedOnly),typeof t.unextractedOnly==`boolean`&&setUnextractedOnly(t.unextractedOnly),typeof e!=`boolean`&&typeof t.compactMode==`boolean`&&m(t.compactMode),",
		},
		{
			old: "zx({filter:s,problemOnly:l,disabledOnly:d,compactMode:p,search:h,page:_,pageSize:tt,regularPageSize:b.regular,compactPageSize:b.compact,sortMode:D}),Vx(p))},[p,d,s,_,tt,b,l,h,D,j])",
			new: "zx({filter:s,problemOnly:l,disabledOnly:d,extractedOnly:extractedOnly,unextractedOnly:unextractedOnly,compactMode:p,search:h,page:_,pageSize:tt,regularPageSize:b.regular,compactPageSize:b.compact,sortMode:D}),Vx(p))},[p,d,s,_,tt,b,l,h,D,j,extractedOnly,unextractedOnly])",
		},
		{
			old: "let st=(0,y.useMemo)(()=>{let e=new Set([`all`]);return I.forEach(t=>{t.type&&e.add(t.type)}),Array.from(e)},[I]),ct=(0,y.useMemo)(()=>I.filter(e=>!(l&&!Vv(e)||d&&e.disabled!==!0)),[d,I,l]),lt=",
			new: "let codexExtractedFilterMatch=e=>String(e?.type??e?.provider??``).trim().toLowerCase()===`codex`&&!!(e?.codex_redeemed||e?.codex_extracted||e?.redeemed),codexUnextractedFilterMatch=e=>String(e?.type??e?.provider??``).trim().toLowerCase()===`codex`&&!codexExtractedFilterMatch(e)&&String(e?.account_status??e?.accountStatus??e?.status??``).trim().toLowerCase()!==`banned`,cardBatchActiveForFilters=String(h||``).trim().startsWith(`__codex_card_batch__=`),st=(0,y.useMemo)(()=>{let e=new Set([`all`]);return I.forEach(t=>{t.type&&e.add(t.type)}),Array.from(e)},[I]),ct=(0,y.useMemo)(()=>I.filter(e=>cardBatchActiveForFilters||!(l&&!Vv(e)||d&&e.disabled!==!0||extractedOnly&&!codexExtractedFilterMatch(e)||unextractedOnly&&!codexUnextractedFilterMatch(e))),[d,I,l,extractedOnly,unextractedOnly,cardBatchActiveForFilters]),lt=",
		},
		{
			old: "dt=h.trim(),ft=(0,y.useMemo)(()=>Yx(dt),[dt]),pt=(0,y.useMemo)(()=>{let e=dt.toLowerCase();return ct.filter(t=>{let n=s===`all`||t.type===s,r=!dt||[t.name,t.type,t.provider].some(t=>{let n=(t||``).toString();return ft?ft.test(n):n.toLowerCase().includes(e)});return n&&r})},[ct,s,dt,ft]),mt=",
			new: "dt=h.trim(),cardBatchSearchMarker=`__codex_card_batch__=`,cardBatchTerms=dt.startsWith(cardBatchSearchMarker)?dt.slice(cardBatchSearchMarker.length).split(`|||`).map(e=>{try{return decodeURIComponent(e).trim().toLowerCase()}catch(t){return e.trim().toLowerCase()}}).filter(Boolean):null,ft=(0,y.useMemo)(()=>cardBatchTerms?null:Yx(dt),[dt,cardBatchTerms]),pt=(0,y.useMemo)(()=>{let e=dt.toLowerCase();return ct.filter(t=>{let n=s===`all`||t.type===s,r=!dt||(cardBatchTerms?cardBatchTerms.some(e=>[t.name,t.id,t.path,t.email,t.account].some(t=>{let n=String(t||``).toLowerCase();return n===e||n.includes(e)})):[t.name,t.type,t.provider].some(t=>{let n=(t||``).toString();return ft?ft.test(n):n.toLowerCase().includes(e)}));return n&&r})},[ct,s,dt,ft,cardBatchTerms]),mt=",
		},
		{
			old: "function ex(e){let{t}=qo(),{file:n,compact:r,selected:i,resolvedTheme:a,disableControls:o,deleting:s,statusUpdating:c,quotaFilterType:l,statusBarCache:u,onShowModels:d,onDownload:f,onOpenPrefixProxyEditor:p,onDelete:m,onToggleStatus:h,onToggleSelect:g}=e,_=",
			new: "function ex(e){let{t}=qo(),{file:n,compact:r,selected:i,resolvedTheme:a,disableControls:o,deleting:s,statusUpdating:c,quotaFilterType:l,statusBarCache:u,onShowModels:d,onDownload:f,onOpenPrefixProxyEditor:p,onDelete:m,onToggleStatus:h,onToggleSelect:g}=e,refreshQuotaNotify=hc(e=>e.showNotification),codexQuotaForCard=np(e=>e.codexQuota[n.name]),setCodexQuotaForCard=np(e=>e.setCodexQuota),refreshCodexQuotaForCard=async()=>{if(o||Kv(n)||n.disabled||codexQuotaForCard?.status===`loading`)return;let e=Xb(`codex`);setCodexQuotaForCard(t=>({...t,[n.name]:e.buildLoadingState()}));try{let r=await e.fetchQuota(n,t);setCodexQuotaForCard(t=>({...t,[n.name]:e.buildSuccessState(r)})),refreshQuotaNotify(t(`auth_files.quota_refresh_success`,{name:n.name}),`success`)}catch(e){let r=e instanceof Error?e.message:t(`common.unknown_error`),i=Ry(e);setCodexQuotaForCard(t=>({...t,[n.name]:Xb(`codex`).buildErrorState(r,i)})),refreshQuotaNotify(t(`auth_files.quota_refresh_failed`,{name:n.name,message:r}),`error`) }},_=",
		},
		{
			old: "!y&&(0,B.jsxs)(`div`,{className:G.statusToggle,children:[(0,B.jsx)(`span`,{className:G.statusToggleLabel,children:t(`auth_files.status_toggle_label`)}),(0,B.jsx)(Sg,{ariaLabel:t(`auth_files.status_toggle_label`),checked:!n.disabled,disabled:o||c[n.name]===!0,onChange:e=>h(n,e)})]})",
			new: "!y&&(0,B.jsxs)(`div`,{className:G.statusToggle,children:[String(n.type||n.provider||``).trim().toLowerCase()===`codex`&&(0,B.jsx)(V,{variant:`secondary`,size:`sm`,className:`auth-file-card-quota-refresh-button`,onClick:()=>void refreshCodexQuotaForCard(),disabled:o||n.disabled||codexQuotaForCard?.status===`loading`,title:`刷新额度`,\"aria-label\":`刷新额度`,children:codexQuotaForCard?.status===`loading`?(0,B.jsx)(p_,{size:13}):(0,B.jsx)(cs,{size:13})}),(0,B.jsx)(`span`,{className:G.statusToggleLabel,children:t(`auth_files.status_toggle_label`)}),(0,B.jsx)(Sg,{ariaLabel:t(`auth_files.status_toggle_label`),checked:!n.disabled,disabled:o||c[n.name]===!0,onChange:e=>h(n,e)})]})",
		},
		{
			old: "P=y?t(`auth_files.type_virtual`)||`虚拟认证文件`:n.disabled?t(`auth_files.health_status_disabled`):t(j?`auth_files.health_status_warning`:A?`auth_files.health_status_healthy`:`auth_files.status_toggle_label`),ee=y?G.stateBadgeVirtual:n.disabled?G.stateBadgeDisabled:j?G.stateBadgeWarning:G.stateBadgeActive;return",
			new: "P=y?t(`auth_files.type_virtual`)||`虚拟认证文件`:n.disabled?t(`auth_files.health_status_disabled`):t(j?`auth_files.health_status_warning`:A?`auth_files.health_status_healthy`:`auth_files.status_toggle_label`),ee=y?G.stateBadgeVirtual:n.disabled?G.stateBadgeDisabled:j?G.stateBadgeWarning:G.stateBadgeActive,te=(n.type||``).toLowerCase()===`codex`,ne=String(n.account_status??n.accountStatus??n.status??``).trim().toLowerCase(),re=ne===`banned`,ie=re?`⛔ 封禁`:`✓ 正常`,ae=re?G.stateBadgeWarning:G.stateBadgeActive;return",
		},
		{
			old: "(0,B.jsx)(`span`,{className:`${G.stateBadge} ${ee}`,children:P})]}),",
			new: "(0,B.jsx)(`span`,{className:`${G.stateBadge} ${ee}`,children:P}),te&&(0,B.jsx)(`span`,{className:`${G.stateBadge} ${ae}`,title:`账号状态`,children:ie})]}),",
		},
		{
			old: "(0,B.jsxs)(`div`,{className:`${G.filterItem} ${G.filterToggleItem}`,children:[(0,B.jsx)(`label`,{children:e(`auth_files.display_options_label`)}),(0,B.jsxs)(`div`,{className:G.filterToggleGroup,children:[(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:l,onChange:e=>{u(e),v(1)},ariaLabel:e(`auth_files.problem_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.problem_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:d,onChange:e=>{f(e),v(1)},ariaLabel:e(`auth_files.disabled_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.disabled_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:p,onChange:e=>m(e),ariaLabel:e(`auth_files.compact_mode_label`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.compact_mode_label`)})})})]})]})",
			new: "(0,B.jsxs)(`div`,{className:`${G.filterItem} ${G.filterToggleItem}`,children:[(0,B.jsx)(`label`,{children:e(`auth_files.display_options_label`)}),(0,B.jsxs)(`details`,{className:`auth-files-display-options-menu`,children:[(0,B.jsxs)(`summary`,{className:`auth-files-display-options-trigger`,children:[(0,B.jsx)(`span`,{children:e(`auth_files.display_options_label`)}),(l||d||extractedOnly||unextractedOnly||p)&&(0,B.jsx)(`span`,{className:`auth-files-display-options-count`,children:(l?1:0)+(d?1:0)+(extractedOnly?1:0)+(unextractedOnly?1:0)+(p?1:0)}),(0,B.jsx)(`span`,{className:`auth-files-display-options-chevron`,children:`⌄`})]}),(0,B.jsxs)(`div`,{className:`${G.filterToggleGroup} auth-files-display-options-list`,children:[(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:l,onChange:e=>{u(e),v(1)},ariaLabel:e(`auth_files.problem_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.problem_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:d,onChange:e=>{f(e),v(1)},ariaLabel:e(`auth_files.disabled_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.disabled_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:unextractedOnly,onChange:e=>{setUnextractedOnly(e),e&&setExtractedOnly(!1),v(1)},ariaLabel:`仅显示未提取凭证`,label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:`仅显示未提取凭证`})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:extractedOnly,onChange:e=>{setExtractedOnly(e),e&&setUnextractedOnly(!1),v(1)},ariaLabel:`仅显示已提取凭证`,label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:`仅显示已提取凭证`})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:p,onChange:e=>m(e),ariaLabel:e(`auth_files.compact_mode_label`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.compact_mode_label`)})})})]})]})]})",
		},
		{
			old: "(0,B.jsx)(V,{variant:`secondary`,size:`sm`,className:`${sb.viewModeButton} ${g===`all`?sb.viewModeButtonActive:``}`,onClick:()=>{m.length>mb?p(!0):d(`all`)},children:i(`auth_files.view_mode_all`)})",
			new: "(0,B.jsx)(V,{variant:`secondary`,size:`sm`,className:`${sb.viewModeButton} ${g===`all`?sb.viewModeButtonActive:``}`,onClick:()=>d(`all`),children:i(`auth_files.view_mode_all`)})",
		},
		{
			old: "Bh=e=>Array.isArray(e)?zh(e.map(e=>String(e??``))):[],Vh=e=>Array.isArray(e)?e.reduce((e,t)=>{if(!t||typeof t!=`object`)return e;let n=t,r=String(n.name??``).trim(),i=typeof n.error==`string`?n.error.trim():typeof n.message==`string`?n.message.trim():``;return!r&&!i||e.push({name:r,error:i||`Unknown error`}),e},[]):[],Hh=(e,t)=>{let n=new Set(t.map(e=>e.name.trim()).filter(Boolean));return n.size===0?[...e]:e.filter(e=>!n.has(e))},Uh=(e,t)=>{let n=Vh(e?.failed),r=Bh(e?.files),i=typeof e?.uploaded==`number`?e.uploaded:r.length>0?r.length:+(t.length===1&&n.length===0),a=r;if(a.length===0&&i>0)if(n.length===0&&i===t.length)a=[...t];else{let e=Hh(t,n);e.length===i&&(a=e)}return{status:typeof e?.status==`string`?e.status:n.length>0?`partial`:`ok`,uploaded:i,files:a,failed:n}},Wh=",
			new: "Bh=e=>Array.isArray(e)?zh(e.map(e=>String(e??``))):[],Vh=e=>Array.isArray(e)?e.reduce((e,t)=>{if(!t||typeof t!=`object`)return e;let n=t,r=String(n.name??``).trim(),i=typeof n.error==`string`?n.error.trim():typeof n.message==`string`?n.message.trim():``;return!r&&!i||e.push({name:r,error:i||`Unknown error`}),e},[]):[],Hh=(e,t)=>{let n=new Set(t.map(e=>e.name.trim()).filter(Boolean));return n.size===0?[...e]:e.filter(e=>!n.has(e))},Uh=(e,t)=>{let n=Vh(e?.failed),r=Bh(e?.files),i=Array.isArray(e?.duplicates)?e.duplicates.map(e=>String(e?.name??e??``).trim()).filter(Boolean):[],a=typeof e?.uploaded==`number`?e.uploaded:r.length>0?r.length:+(t.length===1&&n.length===0),o=r;if(o.length===0&&a>0)if(n.length===0&&a===t.length)o=[...t];else{let e=Hh(t,n);e.length===a&&(o=e)}return{status:typeof e?.status==`string`?e.status:n.length>0?`partial`:i.length>0&&a===0?`duplicate`:`ok`,uploaded:a,files:o,duplicates:i,failed:n}},Wh=",
		},
		{
			old: "let n=await ag.uploadFiles(a),r=n.uploaded;if(r>0){let i=a.length>1?` (${r}/${a.length})`:``;t(`${e(`auth_files.upload_success`)}${i}`,n.failed.length?`warning`:`success`),await A()}if(n.failed.length>0){let r=n.failed.map(e=>`${e.name}: ${e.error}`).join(`; `);t(`${e(`notification.upload_failed`)}: ${r}`,`error`)}",
			new: "let n=await ag.uploadFiles(a),r=n.uploaded,u=Array.isArray(n.duplicates)?n.duplicates.length:0,f=Array.isArray(n.failed)?n.failed.length:0;if(r>0||u>0||f>0){let i=` (成功 ${r} / 重复 ${u} / 失败 ${f} / 总计 ${a.length})`,o=`${e(`auth_files.upload_success`)}${i}`;if(f>0){let s=n.failed.map(e=>`${e.name}: ${e.error}`).join(`; `);o+=`；${e(`notification.upload_failed`)}: ${s}`}t(o,f>0?`warning`:(u>0&&r===0)?`warning`:`success`),await A()}",
		},
	}

	patched := data
	for _, replacement := range replacements {
		patched = bytes.Replace(patched, []byte(replacement.old), []byte(replacement.new), 1)
	}
	return patched
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
.codex-card-admin-sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
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
.codex-card-admin-table-wrap{border:1px solid var(--border-color);border-radius:14px;overflow:auto}
.codex-card-admin-table{width:100%;min-width:1080px;border-collapse:collapse;font-size:13px}
.codex-card-admin-table th,.codex-card-admin-table td{border-bottom:1px solid var(--border-color);padding:11px 12px;text-align:left;vertical-align:top}
.codex-card-admin-table th{color:var(--text-secondary);background:color-mix(in srgb,var(--bg-tertiary) 78%,transparent);font-size:12px;font-weight:800}
.codex-card-admin-table tr:last-child td{border-bottom:0}
.codex-card-admin-table th.select,.codex-card-admin-table td.select{width:48px;text-align:center;vertical-align:middle}
.codex-card-admin-table th.time,.codex-card-admin-table td.time{min-width:142px;white-space:nowrap}
.codex-card-admin-table th.file,.codex-card-admin-table td.file{min-width:150px}
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
.codex-card-admin-bulkbar{border:1px solid var(--border-color);background:var(--bg-secondary);border-radius:14px;align-items:center;justify-content:flex-start;gap:12px;margin:14px 0 14px;padding:14px;display:flex;flex-wrap:wrap}
.codex-card-admin-search{min-width:260px;flex:1 1 420px;max-width:520px}
.codex-card-admin-search .codex-card-admin-input{height:40px}
.codex-card-admin-search .codex-card-admin-search-textarea{height:48px;min-height:48px;max-height:118px;resize:vertical;line-height:1.35;white-space:pre-wrap}
.codex-card-admin-filter{position:relative;min-width:150px;flex:0 0 150px}
.codex-card-admin-filter .codex-card-admin-input{height:40px;padding-right:36px;appearance:none;-webkit-appearance:none;-moz-appearance:none;cursor:pointer;font-size:14px;font-weight:800;line-height:1.2}
.codex-card-admin-filter::after{content:"";position:absolute;right:14px;top:50%;width:12px;height:12px;pointer-events:none;background:url('data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 12 12%22 fill=%22none%22%3E%3Cpath d=%22M3 4.75 6 7.75 9 4.75%22 stroke=%22%238b8680%22 stroke-width=%221.6%22 stroke-linecap=%22round%22 stroke-linejoin=%22round%22/%3E%3C/svg%3E') center/12px 12px no-repeat;transform:translateY(-50%);opacity:.85}
.codex-card-admin-checklabel{color:var(--text-secondary);align-items:center;gap:9px;font-size:13px;font-weight:800;display:inline-flex;cursor:pointer}
.codex-card-admin-selection{color:var(--text-secondary);font-size:13px;font-weight:700;white-space:nowrap;flex:0 0 auto}
.codex-card-admin-bulk-spacer{min-width:16px;flex:1 1 auto}
.codex-card-admin-bulk-actions{align-items:center;justify-content:flex-end;gap:8px;display:flex;flex:0 0 auto;flex-wrap:nowrap}
.codex-card-admin-bulk-actions .codex-card-admin-button{min-height:36px;padding:8px 12px;font-size:13px}
.codex-card-admin-bulk-actions .codex-card-admin-button.icon-only{width:40px;min-width:40px;min-height:40px;padding:0;gap:0;flex:0 0 auto}
.codex-card-admin-bulk-actions .codex-card-admin-button.icon-only svg{width:17px;height:17px;display:block}
.AuthFilesPage-module__filterControls___PfZDU{grid-template-columns:minmax(220px,380px) minmax(86px,132px) minmax(132px,210px) minmax(144px,168px)!important;justify-content:start}
.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__filterItem___Kko4o{width:100%;max-width:100%}
.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__filterItem___Kko4o:nth-child(1){max-width:380px}
.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__filterItem___Kko4o:nth-child(2){max-width:132px}
.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__filterItem___Kko4o:nth-child(3){max-width:210px}
.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__filterItem___Kko4o:nth-child(4){max-width:168px}
.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__pageSizeSelect___yEBvp{width:100%;min-width:0}
.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__sortSelect___4fEjm{width:100%;min-width:0}
.auth-files-display-options-menu{position:relative;width:100%;max-width:168px}
.auth-files-display-options-trigger{box-sizing:border-box;user-select:none;cursor:pointer;list-style:none!important;-webkit-appearance:none;appearance:none;border:1px solid var(--border-color);background:var(--bg-primary);color:var(--text-primary);border-radius:9px;align-items:center;justify-content:space-between;gap:8px;width:100%;height:40px;padding:0 12px;font-size:13px;font-weight:700;line-height:40px;display:flex}
.auth-files-display-options-trigger::-webkit-details-marker{display:none!important}
.auth-files-display-options-trigger::marker{content:"";font-size:0}
.auth-files-display-options-menu[open] .auth-files-display-options-trigger{border-color:var(--primary-color);box-shadow:0 0 0 3px color-mix(in srgb,var(--primary-color) 16%,transparent)}
.auth-files-display-options-count{background:color-mix(in srgb,var(--primary-color) 18%,transparent);color:var(--primary-color);border-radius:999px;min-width:20px;height:20px;place-items:center;padding:0 6px;font-size:11px;font-weight:800;line-height:1;display:inline-grid}
.auth-files-display-options-chevron{display:inline-flex;flex:none;align-items:center;justify-content:center;width:12px;height:12px;background:url('data:image/svg+xml,%3Csvg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 12 12%22 fill=%22none%22%3E%3Cpath d=%22M3 4.75 6 7.75 9 4.75%22 stroke=%22%238b8680%22 stroke-width=%221.6%22 stroke-linecap=%22round%22 stroke-linejoin=%22round%22/%3E%3C/svg%3E') center/12px 12px no-repeat;font-size:0;line-height:0;transition:transform .15s;transform-origin:center}
.auth-files-display-options-menu[open] .auth-files-display-options-chevron{transform:rotate(180deg)}
.auth-files-display-options-list{z-index:40;position:absolute;top:calc(100% + 8px);left:0;right:auto;box-sizing:border-box;width:100%;min-width:0;border:1px solid var(--border-color);background:var(--bg-secondary);border-radius:12px;padding:6px;box-shadow:0 16px 38px color-mix(in srgb,#000 20%,transparent);display:grid!important;gap:3px!important;min-height:0!important}
.auth-files-display-options-list .AuthFilesPage-module__filterToggleCard___N4oxi{border:0;background:transparent;border-radius:8px;padding:0;min-height:0}
.auth-files-display-options-list label[class*="ToggleSwitch-module__root"]{box-sizing:border-box;align-items:center;gap:10px;width:100%;min-height:34px;border-radius:8px;padding:8px 10px;display:flex}
.auth-files-display-options-list label[class*="ToggleSwitch-module__root"]:hover{background:color-mix(in srgb,var(--text-primary) 8%,transparent)}
.auth-files-display-options-list label[class*="ToggleSwitch-module__root"] input{appearance:none!important;opacity:1!important;position:static!important;box-sizing:border-box;flex:none;width:16px!important;height:16px!important;margin:0;border:1px solid var(--border-color);background:var(--bg-primary);border-radius:4px;display:grid;place-content:center}
.auth-files-display-options-list label[class*="ToggleSwitch-module__root"] input:checked{background:var(--primary-color);border-color:var(--primary-color)}
.auth-files-display-options-list label[class*="ToggleSwitch-module__root"] input:checked:after{content:"";width:8px;height:5px;border-left:2px solid #fff;border-bottom:2px solid #fff;transform:rotate(-45deg) translate(1px,-1px)}
.auth-files-display-options-list label[class*="ToggleSwitch-module__root"] [class*="ToggleSwitch-module__track"]{display:none!important}
.auth-files-display-options-list label[class*="ToggleSwitch-module__root"] [class*="ToggleSwitch-module__label"]{color:var(--text-primary);font-size:13px;font-weight:700;line-height:1.35}
.auth-file-card-quota-refresh-button{min-height:30px!important;height:30px!important;border-radius:8px!important;padding:0 8px!important;font-size:12px!important;font-weight:800!important;gap:0!important;white-space:nowrap}
.auth-file-card-quota-refresh-button svg{width:14px;height:14px}
@media (max-width:900px){.codex-card-admin-grid,.codex-card-admin-stats{grid-template-columns:1fr}.codex-card-admin-row,.codex-card-admin-list-head{align-items:stretch;flex-direction:column}.codex-card-admin-bulkbar{align-items:stretch;flex-direction:column}.codex-card-admin-search{min-width:0;max-width:none;flex:auto}.codex-card-admin-selection{align-self:flex-start}.codex-card-admin-bulk-spacer{display:none}.codex-card-admin-filter{width:100%;min-width:0;flex:auto}.codex-card-admin-bulk-actions{align-items:stretch;flex-direction:column}.codex-card-admin-bulk-actions .codex-card-admin-button{width:100%}}
@media (max-width:768px){.AuthFilesPage-module__filterControls___PfZDU{grid-template-columns:1fr!important}.AuthFilesPage-module__filterControls___PfZDU .AuthFilesPage-module__filterItem___Kko4o,.auth-files-display-options-menu{max-width:none}.auth-files-display-options-list{left:0;right:auto;width:100%;min-width:0}}
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
    <div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">-</div><div class="codex-card-admin-stat-label">总提取</div></div>
    <div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">-</div><div class="codex-card-admin-stat-label">今提取</div></div>
  </section>
  <div class="codex-card-admin-grid">
    <section class="codex-card-admin-card">
      <h2>系统生成卡密</h2>
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
      <label class="codex-card-admin-label" for="codexCardImportCodes">待导入卡密</label>
      <textarea class="codex-card-admin-textarea" id="codexCardImportCodes" placeholder="user@example.com---https://mail.lucker.cc.cd/keycode?email=user@example.com&amp;key=et_xxxxxxxxxxxxxxxxxxxxx&#10;CDX-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"></textarea>
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
          <textarea class="codex-card-admin-input codex-card-admin-search-textarea" id="codexCardSearchInput" rows="2" placeholder="搜索卡密、状态、来源、提取时间或兑换文件&#10;批量搜索：一行一个卡密"></textarea>
        </div>
        <span class="codex-card-admin-selection" id="codexCardSelectionStatus">已选择 0 个</span>
        <span class="codex-card-admin-bulk-spacer" aria-hidden="true"></span>
        <div class="codex-card-admin-filter">
          <label class="codex-card-admin-sr-only" for="codexCardStatusFilter">筛选状态</label>
          <select class="codex-card-admin-input" id="codexCardStatusFilter" aria-label="筛选状态">
            <option value="all" selected>全部状态</option>
            <option value="used">已用</option>
            <option value="unused">未用</option>
          </select>
        </div>
        <div class="codex-card-admin-bulk-actions">
          <button class="codex-card-admin-button secondary icon-only" id="codexCardRefreshButton" type="button" title="刷新列表" aria-label="刷新列表">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 12a9 9 0 1 1-2.64-6.36"/><path d="M21 3v6h-6"/></svg>
          </button>
          <button class="codex-card-admin-button secondary icon-only" id="codexCardExportSelectedButton" type="button" disabled title="导出选中" aria-label="导出选中">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/></svg>
          </button>
          <button class="codex-card-admin-button danger icon-only" id="codexCardDeleteSelectedButton" type="button" disabled title="删除选中" aria-label="删除选中">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 6h18"/><path d="M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2"/><path d="M6 6l1 14h10l1-14"/><path d="M10 11v5"/><path d="M14 11v5"/></svg>
          </button>
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

  async function copyTextToClipboard(text) {
    var value = String(text || "");
    if (!value) return false;
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value);
      return true;
    }
    var textarea = document.createElement("textarea");
    textarea.value = value;
    textarea.setAttribute("readonly", "readonly");
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    textarea.style.top = "0";
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    var copied = document.execCommand("copy");
    textarea.remove();
    return copied;
  }

  function extractCardCodeInput(value) {
    var trimmed = String(value || "").trim();
    if (!trimmed) return "";
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
    var markerIndex = trimmed.indexOf("---");
    if (markerIndex >= 0) {
      var suffix = trimmed.slice(markerIndex + 3).trim();
      if (suffix && suffix !== trimmed) candidates.unshift(suffix);
    }
    return candidates;
  }

  function extractCardCodeKeyParam(value) {
    try {
      var parsed = new URL(value, window.location.origin);
      var key = parsed.searchParams.get("key");
      if (key && key.trim()) return key.trim();
    } catch (errParse) {}
    var match = String(value || "").match(/(?:^|[?&#])key=([^&#\s]+)/i);
    if (match && match[1]) {
      try {
        return decodeURIComponent(match[1].replace(/\+/g, " ")).trim();
      } catch (errDecode) {
        return match[1].trim();
      }
    }
    return "";
  }

  function extractCardCodeInputs(text) {
    return String(text || "")
      .replace(/\r\n/g, "\n")
      .replace(/\r/g, "\n")
      .split("\n")
      .map(extractCardCodeInput)
      .filter(Boolean);
  }

  function cardRedeemedAtValue(card) {
    if (!card) return "";
    return card.redeemed_at || card.redeemedAt || "";
  }

  function localDateKey(value) {
    if (!value) return "";
    var d = new Date(value);
    if (Number.isNaN(d.getTime())) return "";
    return d.getFullYear() + "-" + String(d.getMonth() + 1).padStart(2, "0") + "-" + String(d.getDate()).padStart(2, "0");
  }

  function countRedeemedToday(cards) {
    var todayKey = localDateKey(new Date());
    if (!todayKey || !Array.isArray(cards)) return 0;
    return cards.filter(function (card) {
      if (!card || String(card.status || "").trim().toLowerCase() !== "redeemed") return false;
      return localDateKey(cardRedeemedAtValue(card)) === todayKey;
    }).length;
  }

  function renderStats(summary, cards) {
    var root = document.getElementById("codexCardStats");
    if (!root) return;
    var values = summary || {};
    var redeemedToday = values.redeemed_today != null
      ? values.redeemed_today
      : values.today_redeemed != null
        ? values.today_redeemed
        : countRedeemedToday(cards);
    var items = [
      ["total", "总卡密", values.total],
      ["unused", "未使用", values.unused],
      ["redeemed", "总提取", values.redeemed],
      ["redeemed_today", "今提取", redeemedToday]
    ];
    root.innerHTML = items.map(function (item) {
      return '<div class="codex-card-admin-stat"><div class="codex-card-admin-stat-value">' + escapeHTML(item[2] || 0) + '</div><div class="codex-card-admin-stat-label">' + item[1] + '</div></div>';
    }).join("");
  }

  function parseCardSearch(value) {
    var raw = String(value || "");
    var normalized = raw
      .replace(/\r\n/g, "\n")
      .replace(/\r/g, "\n");
    var terms = normalized
      .split("\n")
      .map(extractCardCodeInput)
      .map(function (item) {
        return String(item || "").trim().toLowerCase();
      })
      .filter(Boolean);
    return {
      raw: normalized.trim().toLowerCase(),
      terms: terms,
      batch: normalized.indexOf("\n") >= 0
    };
  }

  function cardSearchHaystack(card) {
    var redeemedAt = cardRedeemedAtValue(card);
    return [
      card.code,
      card.status,
      card.source,
      card.created_at,
      formatDate(card.created_at),
      redeemedAt,
      formatDate(redeemedAt),
      card.redeemed_file,
      card.redeemed_auth_id,
      card.note
    ].join(" ").toLowerCase();
  }

  function cardMatchesSearch(card, search) {
    if (!search || (!search.raw && (!search.terms || search.terms.length === 0))) return true;
    if (!card) return false;
    var code = String(card.code || "").trim().toLowerCase();
    var terms = search.terms || [];
    if (search.batch && terms.length > 0) {
      return terms.some(function (term) {
        return code === term;
      });
    }
    if (terms.length === 1 && code === terms[0]) return true;
    return cardSearchHaystack(card).indexOf(search.raw) >= 0;
  }

  function selectedStatusFilter() {
    var select = document.getElementById("codexCardStatusFilter");
    if (!select) return "all";
    return String(select.value || "all").trim().toLowerCase() || "all";
  }

  function cardMatchesStatus(card, filter) {
    var normalized = String(filter || "all").trim().toLowerCase();
    if (!normalized || normalized === "all") return true;
    if (!card) return false;
    var status = String(card.status || "").trim().toLowerCase();
    if (normalized === "used") {
      return status !== "unused";
    }
    if (normalized === "unused") {
      return status === "unused";
    }
    return true;
  }

  function filteredCards() {
    var input = document.getElementById("codexCardSearchInput");
    var search = parseCardSearch(input ? input.value : "");
    var statusFilter = selectedStatusFilter();
    return (allCards || []).filter(function (card) {
      return cardMatchesSearch(card, search) && cardMatchesStatus(card, statusFilter);
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
      var search = parseCardSearch(searchInput ? searchInput.value : "");
      var statusFilter = selectedStatusFilter();
      var message = search.raw || search.terms.length
        ? search.batch
          ? "没有匹配的卡密，请确认每行一个卡密。"
          : "没有匹配的卡密，请换个关键词。"
        : statusFilter !== "all"
          ? "当前筛选条件下没有卡密。"
          : "还没有卡密，先生成或导入一批。";
      wrap.innerHTML = '<div class="codex-card-admin-empty">' + escapeHTML(message) + '</div>';
      updateSelectionControls();
      return;
    }
    wrap.innerHTML = '<table class="codex-card-admin-table"><thead><tr><th class="select"><input class="codex-card-admin-checkbox" id="codexCardSelectAllTable" type="checkbox" aria-label="全选卡密"></th><th>卡密</th><th>状态</th><th>来源</th><th class="time">创建时间</th><th class="time">提取时间</th><th class="file">兑换文件</th></tr></thead><tbody>' + cards.map(function (card) {
      var status = card.status || "";
      var file = card.redeemed_file ? '<span class="codex-card-admin-code">' + escapeHTML(card.redeemed_file) + '</span>' : "-";
      var redeemedAt = cardRedeemedAtValue(card);
      return '<tr><td class="select"><input class="codex-card-admin-checkbox codex-card-row-checkbox" type="checkbox" value="' + escapeHTML(card.code) + '" aria-label="选择卡密 ' + escapeHTML(card.code) + '"></td><td class="codex-card-admin-code">' + escapeHTML(card.code) + '</td><td><span class="codex-card-admin-pill ' + escapeHTML(status) + '">' + escapeHTML(status) + '</span></td><td>' + escapeHTML(card.source || "-") + '</td><td class="time">' + escapeHTML(formatDate(card.created_at)) + '</td><td class="time">' + escapeHTML(formatDate(redeemedAt)) + '</td><td class="file">' + file + '</td></tr>';
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
      renderStats(data.summary || {}, allCards);
      renderTable(filteredCards());
      updateStatus("codexCardListStatus", "卡密列表已刷新。", "ok");
    } catch (err) {
      updateStatus("codexCardListStatus", err.message || String(err), "error");
    }
  }

  async function exportSelectedCards() {
    var codes = selectedCardCodes();
    if (codes.length === 0) {
      updateStatus("codexCardListStatus", "请先勾选要导出的卡密。", "error");
      return;
    }
    updateStatus("codexCardListStatus", "正在导出选中卡密...", "");
    var data = await apiDownload("/codex-cards/export", {method: "POST", body: JSON.stringify({items: codes})});
    saveBlob(data.blob, data.filename);
    updateStatus("codexCardListStatus", "已导出 " + codes.length + " 个选中卡密。", "ok");
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
          var outputText = codes.join("\n") || JSON.stringify(data, null, 2);
          document.getElementById("codexCardGenerateOutput").textContent = outputText;
          if (outputText) {
            try {
              var copied = await copyTextToClipboard(outputText);
              updateStatus("codexCardGenerateStatus", "已生成 " + codes.length + " 个卡密" + (copied ? "，已复制到剪贴板。" : "，但浏览器未允许自动复制。"), copied ? "ok" : "error");
            } catch (errCopy) {
              updateStatus("codexCardGenerateStatus", "已生成 " + codes.length + " 个卡密，但复制到剪贴板失败。", "error");
            }
          } else {
            updateStatus("codexCardGenerateStatus", "已生成 " + codes.length + " 个卡密。", "ok");
          }
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
        updateStatus("codexCardImportStatus", "正在识别卡密...", "");
        try {
          var rawCodes = document.getElementById("codexCardImportCodes").value || "";
          var codes = extractCardCodeInputs(rawCodes);
          if (codes.length === 0) {
            updateStatus("codexCardImportStatus", "请先输入卡密或邮箱---keycode 链接。", "error");
            return;
          }
          updateStatus("codexCardImportStatus", "已识别 " + codes.length + " 个卡密，正在导入...", "");
          var data = await apiFetch("/codex-cards/import", {method: "POST", body: JSON.stringify({items: codes})});
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
    var statusFilter = document.getElementById("codexCardStatusFilter");
    if (statusFilter) {
      statusFilter.addEventListener("change", applyCardSearch);
    }
    var exportSelectedButton = document.getElementById("codexCardExportSelectedButton");
    if (exportSelectedButton) {
      exportSelectedButton.addEventListener("click", async function () {
        exportSelectedButton.disabled = true;
        try {
          await exportSelectedCards();
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

const authFileCodexStatsScript = `
(function () {
  "use strict";

  var AUTH_FILES_HASH = "#/auth-files";
  var AUTH_KEY = "cli-proxy-auth";
  var SECURE_PREFIX = "enc::v1::";
  var SECURE_NAMESPACE = "cli-proxy-api-webui::secure-storage";
  var STYLE_ID = "auth-file-codex-stats-style";
  var PANEL_ID = "auth-file-codex-stats-panel";
  var observerStarted = false;
  var lastFetchAt = 0;
  var fetching = false;
  var pendingStatsRefresh = false;
  var CARD_BATCH_SEARCH_MARKER = "__codex_card_batch__=";
  var CARD_BATCH_SEARCH_DELIMITER = "|||";
  var CARD_BATCH_SEARCH_INPUT_ID = "auth-file-card-batch-search-input";
  var CARD_BATCH_SEARCH_STATUS_ID = "auth-file-card-batch-search-status";
  var cardBatchSearchTimer = 0;
  var lastCardBatchNotice = "";

  function ensureStatsStyles() {
    if (document.getElementById(STYLE_ID)) return;
    var style = document.createElement("style");
    style.id = STYLE_ID;
    style.textContent = [
      ".auth-file-codex-stats-panel{box-sizing:border-box;width:100%;min-height:76px;margin:0 0 12px;border:1px solid var(--border-color);background:color-mix(in srgb,var(--bg-secondary) 82%,transparent);border-radius:12px;padding:12px 16px;display:flex;align-items:center;gap:12px;flex-wrap:wrap;position:relative;z-index:1;box-shadow:inset 0 1px 0 color-mix(in srgb,#fff 5%,transparent)}",
      ".auth-file-codex-stats-title{color:var(--text-secondary);font-size:12px;font-weight:800;white-space:nowrap;margin-right:2px}",
      ".auth-file-codex-stat{min-width:96px;border:1px solid color-mix(in srgb,var(--border-color) 86%,transparent);background:color-mix(in srgb,var(--bg-primary) 72%,transparent);border-radius:10px;padding:9px 12px;display:flex;align-items:center;justify-content:space-between;gap:10px}",
      ".auth-file-codex-stat-label{color:var(--text-secondary);font-size:12px;font-weight:800;white-space:nowrap}",
      ".auth-file-codex-stat-value{color:var(--text-primary);font-size:20px;font-weight:900;line-height:1;font-variant-numeric:tabular-nums}",
      ".auth-file-codex-stat.normal .auth-file-codex-stat-value,.auth-file-codex-stat.unextracted .auth-file-codex-stat-value{color:var(--success-color)}",
      ".auth-file-codex-stat.banned .auth-file-codex-stat-value{color:var(--error-color)}",
      ".auth-file-codex-stat.extracted .auth-file-codex-stat-value{color:var(--primary-color)}",
      ".auth-files-search-source-hidden{display:none!important}",
      ".auth-files-card-code-search{box-sizing:border-box;width:100%;min-height:50px;max-height:130px;resize:vertical;line-height:1.35;white-space:pre-wrap;font:inherit}",
      ".auth-files-card-code-search-status{color:var(--text-secondary);min-height:18px;margin-top:6px;font-size:12px;font-weight:700;line-height:1.4}",
      ".auth-files-card-code-search-status.ok{color:var(--success-color)}",
      ".auth-files-card-code-search-status.error{color:var(--error-color)}",
      "@media (max-width:1100px){.auth-file-codex-stats-panel{min-height:0}.auth-file-codex-stat{flex:1 1 132px}}"
    ].join("\n");
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

  async function apiFetch(path) {
    var key = managementKey();
    if (!key) throw new Error("missing management key");
    var headers = {
      "Content-Type": "application/json",
      Authorization: "Bearer " + key,
      "X-Management-Key": key
    };
    var resp = await fetch(apiBase() + "/v0/management" + path, {headers: headers, cache: "no-store"});
    if (!resp.ok) throw new Error("HTTP " + resp.status);
    return resp.json();
  }

  function escapeHTML(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function cardCodeInputCandidates(trimmed) {
    var candidates = [trimmed];
    var markerIndex = trimmed.indexOf("---");
    if (markerIndex >= 0) {
      var suffix = trimmed.slice(markerIndex + 3).trim();
      if (suffix && suffix !== trimmed) candidates.unshift(suffix);
    }
    return candidates;
  }

  function extractCardCodeKeyParam(value) {
    try {
      var parsed = new URL(value, window.location.origin);
      var key = parsed.searchParams.get("key");
      if (key && key.trim()) return key.trim();
    } catch (errParse) {}
    var match = String(value || "").match(/(?:^|[?&#])key=([^&#\s]+)/i);
    if (match && match[1]) {
      try {
        return decodeURIComponent(match[1].replace(/\+/g, " ")).trim();
      } catch (errDecode) {
        return match[1].trim();
      }
    }
    return "";
  }

  function extractCardCodeInput(value) {
    var trimmed = String(value || "").trim();
    if (!trimmed) return "";
    var candidates = cardCodeInputCandidates(trimmed);
    for (var i = 0; i < candidates.length; i += 1) {
      var key = extractCardCodeKeyParam(candidates[i]);
      if (key) return key;
    }
    return trimmed;
  }

  function normalizeCardCodeLookup(value) {
    return String(extractCardCodeInput(value) || "").trim().toLowerCase();
  }

  function parseCardBatchSearchInput(value) {
    var raw = String(value || "").replace(/\r\n/g, "\n").replace(/\r/g, "\n");
    var terms = raw.split("\n").map(function (line) {
      return {
        raw: String(line || "").trim(),
        code: extractCardCodeInput(line)
      };
    }).filter(function (item) {
      return item.code;
    });
    return {
      raw: raw,
      terms: terms,
      batch: raw.indexOf("\n") >= 0 && terms.length > 0
    };
  }

  function setNativeFieldValue(field, value) {
    if (!field) return;
    var previousValue = field.value;
    var proto = field.tagName === "TEXTAREA" ? window.HTMLTextAreaElement.prototype : window.HTMLInputElement.prototype;
    var descriptor = Object.getOwnPropertyDescriptor(proto, "value");
    if (descriptor && descriptor.set) descriptor.set.call(field, value);
    else field.value = value;
    if (field._valueTracker) {
      field._valueTracker.setValue(previousValue);
    }
    field.dispatchEvent(new Event("input", {bubbles: true}));
    field.dispatchEvent(new Event("change", {bubbles: true}));
  }

  function setCardBatchStatus(message, type) {
    var status = document.getElementById(CARD_BATCH_SEARCH_STATUS_ID);
    if (!status) return;
    status.textContent = message || "";
    status.className = "auth-files-card-code-search-status" + (type ? " " + type : "");
  }

  function encodeCardBatchSearchTerms(terms) {
    var unique = [];
    var seen = new Set();
    (terms || []).forEach(function (term) {
      var value = String(term || "").trim();
      var key = value.toLowerCase();
      if (!value || seen.has(key)) return;
      seen.add(key);
      unique.push(encodeURIComponent(value));
    });
    return CARD_BATCH_SEARCH_MARKER + unique.join(CARD_BATCH_SEARCH_DELIMITER);
  }

  function addCardBatchTarget(targets, value) {
    var trimmed = String(value || "").trim();
    if (trimmed) targets.push(trimmed);
  }

  function cardBatchNoticeLine(input, status) {
    var code = String(input || "").trim();
    return (code || "-") + "：" + status;
  }

  function showCardBatchNotice(lines) {
    if (!lines || lines.length === 0) return;
    var message = "以下卡密没有对应已提取认证文件：\n" + lines.join("\n");
    if (message === lastCardBatchNotice) return;
    lastCardBatchNotice = message;
    window.alert(message);
  }

  async function resolveCardBatchSearch(parsed, sourceInput) {
    setCardBatchStatus("正在按卡密查找对应的认证文件...", "");
    var data = await apiFetch("/codex-cards");
    var cards = Array.isArray(data && data.cards) ? data.cards : [];
    var byCode = new Map();
    cards.forEach(function (card) {
      var key = normalizeCardCodeLookup(card && card.code);
      if (key && !byCode.has(key)) byCode.set(key, card);
    });
    var targets = [];
    var notices = [];
    (parsed.terms || []).forEach(function (item) {
      var lookup = normalizeCardCodeLookup(item.code);
      var card = byCode.get(lookup);
      var displayCode = item.code || item.raw;
      if (!card) {
        notices.push(cardBatchNoticeLine(displayCode, "未找到"));
        return;
      }
      displayCode = card.code || displayCode;
      var status = String(card.status || "").trim().toLowerCase();
      if (status === "unused") {
        notices.push(cardBatchNoticeLine(displayCode, "未使用"));
        return;
      }
      if (status !== "redeemed") {
        notices.push(cardBatchNoticeLine(displayCode, status || "未使用"));
        return;
      }
      var target = card.redeemed_file || card.redeemedFile || card.redeemed_auth_id || card.redeemedAuthID || "";
      if (target) {
        addCardBatchTarget(targets, target);
      } else {
        notices.push(cardBatchNoticeLine(displayCode, "已提取但未记录认证文件"));
      }
    });
    var encodedSearch = encodeCardBatchSearchTerms(targets);
    var matchedCount = encodedSearch === CARD_BATCH_SEARCH_MARKER
      ? 0
      : encodedSearch.slice(CARD_BATCH_SEARCH_MARKER.length).split(CARD_BATCH_SEARCH_DELIMITER).filter(Boolean).length;
    setNativeFieldValue(sourceInput, encodedSearch);
    if (matchedCount > 0) {
      setCardBatchStatus("已按卡密匹配到 " + matchedCount + " 个认证文件。", notices.length > 0 ? "error" : "ok");
    } else {
      setCardBatchStatus("没有匹配到已提取认证文件。", "error");
    }
    showCardBatchNotice(notices);
  }

  function scheduleCardBatchSearch(helper, sourceInput) {
    window.clearTimeout(cardBatchSearchTimer);
    cardBatchSearchTimer = window.setTimeout(function () {
      var parsed = parseCardBatchSearchInput(helper.value);
      if (!parsed.raw.trim()) {
        lastCardBatchNotice = "";
        setCardBatchStatus("", "");
        setNativeFieldValue(sourceInput, "");
        return;
      }
      if (!parsed.batch) {
        lastCardBatchNotice = "";
        setCardBatchStatus("", "");
        setNativeFieldValue(sourceInput, parsed.raw.trim());
        return;
      }
      resolveCardBatchSearch(parsed, sourceInput).catch(function (err) {
        setCardBatchStatus((err && err.message) || String(err), "error");
      });
    }, 450);
  }

  function findAuthFilesSearchInput() {
    var controls = findFilterControls();
    if (!controls) return null;
    var labels = Array.from(controls.querySelectorAll("label"));
    for (var i = 0; i < labels.length; i += 1) {
      var label = labels[i];
      if ((label.textContent || "").indexOf("搜索配置文件") < 0) continue;
      var box = label.parentElement || controls;
      var input = box.querySelector("input:not(.auth-files-card-code-search),textarea:not(.auth-files-card-code-search)");
      if (input) return input;
    }
    return controls.querySelector("input[placeholder*='输入名称']:not(.auth-files-card-code-search)");
  }

  function ensureCardBatchSearch() {
    if (window.location.hash !== AUTH_FILES_HASH) return;
    ensureStatsStyles();
    var sourceInput = findAuthFilesSearchInput();
    if (!sourceInput || !sourceInput.parentElement) return;
    sourceInput.classList.add("auth-files-search-source-hidden");
    sourceInput.setAttribute("aria-hidden", "true");
    sourceInput.setAttribute("tabindex", "-1");
    var parent = sourceInput.parentElement;
    var helper = parent.querySelector("#" + CARD_BATCH_SEARCH_INPUT_ID);
    if (!helper) {
      helper = document.createElement("textarea");
      helper.id = CARD_BATCH_SEARCH_INPUT_ID;
      helper.rows = 2;
      helper.className = String(sourceInput.className || "").replace("auth-files-search-source-hidden", "") + " auth-files-card-code-search";
      helper.placeholder = "输入名称、类型或提供方关键字；也可粘贴卡密，一行一个";
      if (sourceInput.value && sourceInput.value.indexOf(CARD_BATCH_SEARCH_MARKER) !== 0) helper.value = sourceInput.value;
      sourceInput.insertAdjacentElement("afterend", helper);
    }
    var status = parent.querySelector("#" + CARD_BATCH_SEARCH_STATUS_ID);
    if (!status) {
      status = document.createElement("div");
      status.id = CARD_BATCH_SEARCH_STATUS_ID;
      status.className = "auth-files-card-code-search-status";
      helper.insertAdjacentElement("afterend", status);
    }
    if (helper.dataset.cardBatchBound === "1") return;
    helper.dataset.cardBatchBound = "1";
    helper.addEventListener("input", function () {
      scheduleCardBatchSearch(helper, sourceInput);
    });
  }

  function numberValue(value) {
    var n = Number(value);
    return Number.isFinite(n) && n >= 0 ? Math.floor(n) : 0;
  }

  function isCodexFile(file) {
    var provider = String(file && (file.type || file.provider) || "").trim().toLowerCase();
    return provider === "codex";
  }

  function isBannedFile(file) {
    var status = String(file && (file.account_status || file.accountStatus || file.status) || "").trim().toLowerCase();
    return status === "banned";
  }

  function isExtractedFile(file) {
    return !!(file && (file.codex_redeemed || file.codex_extracted || file.redeemed));
  }

  function statsFromFiles(files) {
    var stats = {total: 0, normal: 0, banned: 0, unextracted: 0, extracted: 0};
    (Array.isArray(files) ? files : []).forEach(function (file) {
      if (!isCodexFile(file)) return;
      stats.total += 1;
      var banned = isBannedFile(file);
      if (banned) {
        stats.banned += 1;
      } else {
        stats.normal += 1;
      }
      if (isExtractedFile(file)) stats.extracted += 1;
      else if (!banned) stats.unextracted += 1;
    });
    return stats;
  }

  function normalizeStats(data) {
    var raw = data && (data.codex_auth_stats || data.codexAuthStats);
    if (!raw) return statsFromFiles(data && data.files);
    return {
      total: numberValue(raw.total),
      normal: numberValue(raw.normal),
      banned: numberValue(raw.banned),
      unextracted: raw.unextracted != null ? numberValue(raw.unextracted) : numberValue(raw.unredeemed),
      extracted: raw.extracted != null ? numberValue(raw.extracted) : numberValue(raw.redeemed)
    };
  }

  function findFilterControls() {
    return Array.from(document.querySelectorAll("div")).find(function (node) {
      var cls = String(node.className || "");
      var text = node.innerText || "";
      return cls.indexOf("filterControls___") >= 0 && text.indexOf("搜索配置文件") >= 0 && text.indexOf("显示选项") >= 0;
    }) || null;
  }

  function renderPanel(panel, stats, loading) {
    var items = [
      ["账号总数", stats.total, "total", ""],
      ["正常", stats.normal, "normal", ""],
      ["封禁", stats.banned, "banned", ""],
      ["未提取", stats.unextracted, "unextracted", "未提取=状态正常且尚未分配给用户"],
      ["已提取", stats.extracted, "extracted", "已提取=已分配给用户"]
    ];
    panel.innerHTML = '<div class="auth-file-codex-stats-title">Codex账号统计' + (loading ? ' · 更新中' : '') + '</div>' + items.map(function (item) {
      return '<div class="auth-file-codex-stat ' + item[2] + '"' + (item[3] ? ' title="' + escapeHTML(item[3]) + '"' : '') + '><span class="auth-file-codex-stat-label">' + escapeHTML(item[0]) + '</span><span class="auth-file-codex-stat-value">' + escapeHTML(item[1]) + '</span></div>';
    }).join("");
  }

  function ensurePanel() {
    ensureStatsStyles();
    if (window.location.hash !== AUTH_FILES_HASH) {
      var existing = document.getElementById(PANEL_ID);
      if (existing) existing.remove();
      lastFetchAt = 0;
      return null;
    }
    var controls = findFilterControls();
    if (!controls) return null;
    var panel = document.getElementById(PANEL_ID);
    if (!panel) {
      panel = document.createElement("div");
      panel.id = PANEL_ID;
      panel.className = "auth-file-codex-stats-panel";
      panel.style.gridColumn = "1 / -1";
      panel.dataset.codexStatsLoaded = "0";
      renderPanel(panel, {total: 0, normal: 0, banned: 0, unextracted: 0, extracted: 0}, true);
      var parent = controls.parentElement;
      if (parent && parent !== controls) {
        parent.insertBefore(panel, controls);
      } else {
        controls.insertBefore(panel, controls.firstChild);
      }
    }
    ensureCardBatchSearch();
    return panel;
  }

  async function refreshStats(force) {
    var panel = ensurePanel();
    if (!panel) return;
    if (fetching) {
      pendingStatsRefresh = true;
      return;
    }
    var now = Date.now();
    var needsInitialLoad = panel.dataset.codexStatsLoaded !== "1";
    if (!force && !needsInitialLoad && now - lastFetchAt < 4000) return;
    fetching = true;
    pendingStatsRefresh = false;
    lastFetchAt = now;
    try {
      var data = await apiFetch("/auth-files?is_webui=1");
      var currentPanel = ensurePanel();
      if (currentPanel) {
        renderPanel(currentPanel, normalizeStats(data), false);
        currentPanel.dataset.codexStatsLoaded = "1";
      }
    } catch (err) {
      var errorPanel = ensurePanel();
      if (errorPanel) {
        errorPanel.innerHTML = '<div class="auth-file-codex-stats-title">Codex账号统计</div><div class="auth-file-codex-stat banned"><span class="auth-file-codex-stat-label">统计加载失败</span><span class="auth-file-codex-stat-value">!</span></div>';
        errorPanel.dataset.codexStatsLoaded = "1";
      }
    } finally {
      fetching = false;
      if (pendingStatsRefresh && window.location.hash === AUTH_FILES_HASH) {
        pendingStatsRefresh = false;
        setTimeout(function () { refreshStats(true); }, 50);
      }
    }
  }

  function closeDisplayOptionsMenus(event) {
    var target = event && event.target;
    document.querySelectorAll(".auth-files-display-options-menu[open]").forEach(function (menu) {
      if (target && menu.contains(target)) return;
      menu.removeAttribute("open");
    });
  }

  function closeAllDisplayOptionsMenus() {
    document.querySelectorAll(".auth-files-display-options-menu[open]").forEach(function (menu) {
      menu.removeAttribute("open");
    });
  }

  function bootAuthFileStats() {
    ensurePanel();
    ensureCardBatchSearch();
    refreshStats(false);
    if (observerStarted) return;
    observerStarted = true;
    var observer = new MutationObserver(function () {
      ensurePanel();
      ensureCardBatchSearch();
      refreshStats(false);
    });
    observer.observe(document.body, {childList: true, subtree: true});
    window.addEventListener("hashchange", function () {
      lastFetchAt = 0;
      setTimeout(function () { refreshStats(true); }, 100);
    });
    window.addEventListener("cli-proxy-auth-files-updated", function () {
      setTimeout(function () { refreshStats(true); }, 150);
    });
    window.addEventListener("focus", function () { refreshStats(false); });
    document.addEventListener("pointerdown", closeDisplayOptionsMenus, true);
    document.addEventListener("keydown", function (event) {
      if (event && event.key === "Escape") closeAllDisplayOptionsMenus();
    }, true);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () { setTimeout(bootAuthFileStats, 120); });
  } else {
    setTimeout(bootAuthFileStats, 120);
  }
})();
`
