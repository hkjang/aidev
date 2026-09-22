// A task-local HTTPS smart-Git server for real browser acceptance tests. It
// creates its own temporary bare repository and never opens a project remote.
package main

import (
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitServer "github.com/go-git/go-git/v5/plumbing/transport/server"
)

func main() {
	dir, e := os.MkdirTemp("", "madi-git-http-fixture-")
	if e != nil {
		panic(e)
	}
	defer os.RemoveAll(dir)
	if _, e = git.PlainInit(dir, true); e != nil {
		panic(e)
	}
	endpoint, e := transport.NewEndpoint(dir)
	if e != nil {
		panic(e)
	}
	server := gitServer.NewServer(gitServer.DefaultLoader)
	var mu sync.Mutex
	httpServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		username, password, ok := r.BasicAuth()
		if !ok || username != "fixture" || password != "Git-Browser-Fixture-Only!" {
			w.Header().Set("WWW-Authenticate", "Basic realm=fixture")
			w.WriteHeader(401)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/repo.git/") {
			http.NotFound(w, r)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64<<20)
		service := r.URL.Query().Get("service")
		if service == "" {
			service = strings.TrimPrefix(r.URL.Path, "/repo.git/")
		}
		switch service {
		case "git-upload-pack":
			session, e := server.NewUploadPackSession(endpoint, nil)
			if e != nil {
				http.Error(w, "fixture session", 500)
				return
			}
			defer session.Close()
			if r.Method == "GET" {
				refs, e := session.AdvertisedReferencesContext(r.Context())
				if e != nil {
					http.Error(w, "fixture refs", 500)
					return
				}
				w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
				enc := pktline.NewEncoder(w)
				_ = enc.EncodeString("# service=git-upload-pack\n")
				_ = enc.Flush()
				_ = refs.Encode(w)
				return
			}
			request := packp.NewUploadPackRequest()
			if e = request.Decode(r.Body); e != nil {
				http.Error(w, "fixture request", 400)
				return
			}
			response, e := session.UploadPack(r.Context(), request)
			if e != nil {
				http.Error(w, "fixture pack", 500)
				return
			}
			defer response.Close()
			w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
			_ = response.Encode(w)
		case "git-receive-pack":
			session, e := server.NewReceivePackSession(endpoint, nil)
			if e != nil {
				http.Error(w, "fixture session", 500)
				return
			}
			defer session.Close()
			if r.Method == "GET" {
				refs, e := session.AdvertisedReferencesContext(r.Context())
				if e != nil {
					http.Error(w, "fixture refs", 500)
					return
				}
				w.Header().Set("Content-Type", "application/x-git-receive-pack-advertisement")
				enc := pktline.NewEncoder(w)
				_ = enc.EncodeString("# service=git-receive-pack\n")
				_ = enc.Flush()
				_ = refs.Encode(w)
				return
			}
			request := packp.NewReferenceUpdateRequest()
			if e = request.Decode(r.Body); e != nil {
				http.Error(w, "fixture request", 400)
				return
			}
			response, e := session.ReceivePack(r.Context(), request)
			if e != nil {
				http.Error(w, "fixture update", 500)
				return
			}
			w.Header().Set("Content-Type", "application/x-git-receive-pack-result")
			_ = response.Encode(w)
		default:
			http.NotFound(w, r)
		}
	}))
	defer httpServer.Close()
	raw, _ := json.Marshal(map[string]any{"url": httpServer.URL + "/repo.git", "ca_pem": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: httpServer.Certificate().Raw}))})
	fmt.Println(string(raw))
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
