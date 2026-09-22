package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"golang.org/x/crypto/ssh"
)

func gitSSHTestKey(t *testing.T) (ssh.Signer, string) {
	t.Helper()
	_, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	signer, e := ssh.NewSignerFromKey(key)
	if e != nil {
		t.Fatal(e)
	}
	der, e := x509.MarshalPKCS8PrivateKey(key)
	if e != nil {
		t.Fatal(e)
	}
	return signer, string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func TestGitSyncHTTPSPinsHostCAAndForbidsCredentialRedirect(t *testing.T) {
	var reached atomic.Bool
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Store(true); w.WriteHeader(200) }))
	defer other.Close()
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Error("expected configured credentials at original host")
		}
		w.Header().Set("Location", other.URL+"/repo.git/info/refs")
		w.WriteHeader(302)
	}))
	defer origin.Close()
	cfg := map[string]any{"url": origin.URL + "/repo.git", "branch": "main", "prefix": "madi", "username": "git", "password": "fake-test-token"}
	remote, e := newGitSyncRemote(context.Background(), cfg, []string{"127.0.0.1"}, true)
	if e != nil {
		t.Fatal(e)
	}
	session, e := remote.Transport.NewUploadPackSession(remote.Endpoint, remote.Auth)
	if e == nil {
		_, e = session.AdvertisedReferencesContext(context.Background())
		session.Close()
	}
	remote.Close()
	if e == nil {
		t.Fatal("untrusted CA accepted")
	}
	cfg["ca_pem"] = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: origin.Certificate().Raw}))
	remote, e = newGitSyncRemote(context.Background(), cfg, []string{"127.0.0.1"}, true)
	if e != nil {
		t.Fatal(e)
	}
	defer remote.Close()
	session, e = remote.Transport.NewUploadPackSession(remote.Endpoint, remote.Auth)
	if e == nil {
		_, e = session.AdvertisedReferencesContext(context.Background())
		session.Close()
	}
	if e == nil {
		t.Fatal("redirect accepted")
	}
	if reached.Load() {
		t.Fatal("Git redirect reached another origin")
	}
	reader := &gitSyncBodyLimit{ReadCloser: io.NopCloser(strings.NewReader("12345")), remaining: 3}
	if _, e = io.ReadAll(reader); e == nil {
		t.Fatal("Git response size cap bypass")
	}
}

func TestGitSyncSSHPinsKeyAndIgnoresAmbientProxy(t *testing.T) {
	t.Setenv("ALL_PROXY", "socks5://127.0.0.1:1")
	serverSigner, _ := gitSSHTestKey(t)
	clientSigner, privateKey := gitSSHTestKey(t)
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	serverConfig := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if string(key.Marshal()) != string(clientSigner.PublicKey().Marshal()) {
			return nil, io.EOF
		}
		return nil, nil
	}}
	serverConfig.AddHostKey(serverSigner)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, e := listener.Accept()
			if e != nil {
				return
			}
			go func() {
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(10 * time.Second))
				server, channels, requests, e := ssh.NewServerConn(conn, serverConfig)
				if e != nil {
					return
				}
				defer server.Close()
				go ssh.DiscardRequests(requests)
				for request := range channels {
					channel, requests, e := request.Accept()
					if e != nil {
						return
					}
					for request := range requests {
						var command struct{ Command string }
						if request.Type != "exec" || ssh.Unmarshal(request.Payload, &command) != nil || command.Command != "git-upload-pack '/repo.git'" {
							request.Reply(false, nil)
							continue
						}
						request.Reply(true, nil)
						adv := packp.NewAdvRefs()
						adv.References["refs/heads/main"] = plumbing.NewHash(strings.Repeat("a", 40))
						adv.Encode(channel)
						io.Copy(io.Discard, channel)
						channel.Close()
					}
				}
			}()
		}
	}()
	cfg := map[string]any{"url": "ssh://git@" + listener.Addr().String() + "/repo.git", "branch": "main", "prefix": "madi", "private_key": privateKey, "host_key": string(ssh.MarshalAuthorizedKey(serverSigner.PublicKey()))}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	remote, e := newGitSyncRemote(ctx, cfg, []string{"127.0.0.1"}, true)
	if e != nil {
		t.Fatal(e)
	}
	session, e := remote.Transport.NewUploadPackSession(remote.Endpoint, remote.Auth)
	if e != nil {
		remote.Close()
		t.Fatal(e)
	}
	refs, e := session.AdvertisedReferencesContext(ctx)
	session.Close()
	remote.Close()
	if e != nil || refs.References["refs/heads/main"].IsZero() {
		t.Fatal("pinned SSH refs failed", e)
	}
	wrong, _ := gitSSHTestKey(t)
	cfg["host_key"] = string(ssh.MarshalAuthorizedKey(wrong.PublicKey()))
	remote, e = newGitSyncRemote(ctx, cfg, []string{"127.0.0.1"}, true)
	if e != nil {
		t.Fatal(e)
	}
	defer remote.Close()
	session, e = remote.Transport.NewUploadPackSession(remote.Endpoint, remote.Auth)
	if e == nil {
		_, e = session.AdvertisedReferencesContext(ctx)
		session.Close()
	}
	if e == nil {
		t.Fatal("changed SSH host key accepted")
	}
	listener.Close()
	<-done
}
