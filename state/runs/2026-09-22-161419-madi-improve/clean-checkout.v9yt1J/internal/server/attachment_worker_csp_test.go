package server

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"testing/fstest"
)

func TestBrowserAttachmentWorkerIsolation(t *testing.T) {
	if os.Getenv("MADI_BROWSER_ATTACHMENT_EXTRACTION") != "1" {
		t.Skip("MADI_BROWSER_ATTACHMENT_EXTRACTION=1 enables real worker CSP validation")
	}
	seed, _ := integrationTestServer(t)
	worker := []byte(`const bytes=new Uint8Array([0,97,115,109,1,0,0,0]);let wasm=false,dynamic=false,external=false;try{await WebAssembly.compile(bytes);wasm=true}catch{}try{new Function('return 1')();dynamic=true}catch{}try{await fetch('https://madi-csp-test.invalid/');external=true}catch{}self.postMessage({wasm,dynamic,external});`)
	probe := []byte(`async function worker(url){return await new Promise((resolve,reject)=>{const w=new Worker(url,{type:'module'});w.onmessage=e=>{resolve(e.data);w.terminate()};w.onerror=reject})}const result={pdf:await worker('/assets/pdf.worker.min-csp.mjs'),ordinary:await worker('/assets/ordinary-worker.mjs'),pageWasm:false};try{await WebAssembly.compile(new Uint8Array([0,97,115,109,1,0,0,0]));result.pageWasm=true}catch{}document.getElementById('result').textContent=JSON.stringify(result);`)
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte(`<html><body><pre id="result"></pre><script type="module" src="/assets/probe.mjs"></script></body></html>`)}, "assets/probe.mjs": &fstest.MapFile{Data: probe}, "assets/pdf.worker.min-csp.mjs": &fstest.MapFile{Data: worker}, "assets/ordinary-worker.mjs": &fstest.MapFile{Data: worker}, "plugin-sandbox.html": &fstest.MapFile{Data: []byte(`<html>plugin isolation</html>`)}}
	app, e := New(context.Background(), seed.DB, seed.EncryptionKey, "test", "admin@example.test", "Integration-Test-Password-2026!", assets)
	if e != nil {
		t.Fatal(e)
	}
	endpoint := httptest.NewServer(app)
	defer endpoint.Close()
	defer app.CloseCollaboration()
	cmd := exec.CommandContext(t.Context(), "node", "../../tests/attachment-worker-csp.mjs")
	cmd.Env = append(os.Environ(), "MADI_BASE_URL="+endpoint.URL)
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("worker CSP: %v\n%s", e, output)
	}
	t.Log(string(output))
}
