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

func TestPatchAuthFilesDisplayOptionsDropdown(t *testing.T) {
	input := []byte("(0,B.jsxs)(`div`,{className:`${G.filterItem} ${G.filterToggleItem}`,children:[(0,B.jsx)(`label`,{children:e(`auth_files.display_options_label`)}),(0,B.jsxs)(`div`,{className:G.filterToggleGroup,children:[(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:l,onChange:e=>{u(e),v(1)},ariaLabel:e(`auth_files.problem_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.problem_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:d,onChange:e=>{f(e),v(1)},ariaLabel:e(`auth_files.disabled_filter_only`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.disabled_filter_only`)})})}),(0,B.jsx)(`div`,{className:G.filterToggleCard,children:(0,B.jsx)(Sg,{checked:p,onChange:e=>m(e),ariaLabel:e(`auth_files.compact_mode_label`),label:(0,B.jsx)(`span`,{className:G.filterToggleLabel,children:e(`auth_files.compact_mode_label`)})})})]})]})")

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "auth-files-display-options-menu")
	assertContains(t, patched, "auth-files-display-options-trigger")
	assertContains(t, patched, "auth-files-display-options-list")
	assertContains(t, patched, "children:(l?1:0)+(d?1:0)+(p?1:0)")
	assertNotContains(t, patched, "className:G.filterToggleGroup,children:[")
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

func TestCodexCardManagementPanelExtractsKeyFromTokenCodeLinks(t *testing.T) {
	script := []byte(codexCardManagementPanelScript)

	assertContains(t, script, "一行一个卡密或 token-code 链接")
	assertContains(t, script, "function extractCardCodeInput")
	assertContains(t, script, "searchParams.get(\"key\")")
	assertContains(t, script, "function extractCardCodeInputs")
	assertContains(t, script, "JSON.stringify({items: codes})")
}

func TestCodexCardManagementPanelIncludesAuthFilesFilterStyles(t *testing.T) {
	script := []byte(codexCardManagementPanelScript)

	assertContains(t, script, ".auth-files-display-options-menu")
	assertContains(t, script, ".auth-files-display-options-list")
	assertContains(t, script, "ToggleSwitch-module__root")
	assertContains(t, script, "grid-template-columns:minmax(220px,380px)")
	assertContains(t, script, "left:0;right:auto")
	assertContains(t, script, "width:100%;min-width:0")
	assertContains(t, script, "::-webkit-details-marker{display:none!important}")
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
