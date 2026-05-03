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
			"(0,B.jsx)(V,{variant:`secondary`,size:`sm`,className:`${sb.viewModeButton} ${g===`all`?sb.viewModeButtonActive:``}`,onClick:()=>{m.length>mb?p(!0):d(`all`)},children:i(`auth_files.view_mode_all`)})",
	)

	patched := patchQuotaManagementPanel(input)

	assertContains(t, patched, "var pb=25,mb=1e3,")
	assertContains(t, patched, "[q,z]=(0,y.useState)(``)")
	assertContains(t, patched, "title:i(`auth_files.page_size_label`)")
	assertContains(t, patched, "onClick:()=>d(`all`)")
	assertContains(t, patched, "O(t,`all`,E)")
	assertContains(t, patched, ";let qn=Math.min(c*3,pb)")
	assertNotContains(t, patched, "onClick:()=>{m.length>mb?p(!0):d(`all`)}")
	assertNotContains(t, patched, "let t=g===`all`?`all`:`page`")
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
