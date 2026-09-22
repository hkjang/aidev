package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport"
	gitHTTP "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitSSH "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

type gitSyncRemote struct {
	Transport transport.Transport
	Endpoint  *transport.Endpoint
	Auth      transport.AuthMethod
	Close     func()
	AllowPack func()
}
type gitSyncNetwork struct {
	address     string
	read, write atomic.Int64
	phase       atomic.Int64
	mu          sync.Mutex
	connections []net.Conn
	closed      bool
	deadline    time.Time
}
type gitSyncConn struct {
	net.Conn
	budget *gitSyncNetwork
}

func (c *gitSyncConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	if c.budget.read.Add(-int64(n)) < 0 || c.budget.phase.Add(-int64(n)) < 0 {
		return 0, errGitSyncLimit
	}
	return n, e
}
func (c *gitSyncConn) Write(p []byte) (int, error) {
	if c.budget.write.Add(-int64(len(p))) < 0 {
		return 0, errGitSyncLimit
	}
	return c.Conn.Write(p)
}
func (n *gitSyncNetwork) Dial(network, address string) (net.Conn, error) {
	return n.DialContext(context.Background(), network, address)
}
func (n *gitSyncNetwork) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if network != "tcp" || address != n.address {
		return nil, errors.New("Git 연결 대상이 고정 주소와 다릅니다")
	}
	n.mu.Lock()
	closed := n.closed
	n.mu.Unlock()
	if closed {
		return nil, net.ErrClosed
	}
	c, e := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", n.address)
	if e != nil {
		return nil, errors.New("Git 서버에 연결하지 못했습니다")
	}
	c.SetDeadline(n.deadline)
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		c.Close()
		return nil, net.ErrClosed
	}
	n.connections = append(n.connections, c)
	return &gitSyncConn{Conn: c, budget: n}, nil
}
func (n *gitSyncNetwork) close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closed = true
	for _, c := range n.connections {
		c.Close()
	}
	n.connections = nil
}

var gitSyncSSHNetworks sync.Map

type gitSyncHTTPTransport struct{ http.RoundTripper }
type gitSyncBodyLimit struct {
	io.ReadCloser
	remaining int64
}

func (r *gitSyncBodyLimit) Read(p []byte) (int, error) {
	if r.remaining < 0 {
		return 0, errGitSyncLimit
	}
	if int64(len(p)) > r.remaining+1 {
		p = p[:r.remaining+1]
	}
	n, e := r.ReadCloser.Read(p)
	r.remaining -= int64(n)
	if r.remaining < 0 {
		return 0, errGitSyncLimit
	}
	return n, e
}
func (t gitSyncHTTPTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, e := t.RoundTripper.RoundTrip(r)
	if e != nil {
		return nil, e
	}
	limit := gitSyncMaxPackBytes + (1 << 20)
	if strings.HasSuffix(r.URL.Path, "/info/refs") {
		limit = 1 << 20
	}
	response.Body = &gitSyncBodyLimit{ReadCloser: response.Body, remaining: limit}
	return response, nil
}

func init() {
	// This service never inherits ~/.ssh/config or SSH_AUTH_SOCK. A dedicated
	// direct proxy adapter also avoids x/net/proxy's ALL_PROXY environment path.
	gitSSH.DefaultSSHConfig = nil
	proxy.RegisterDialerType("madi-git-direct", func(u *url.URL, _ proxy.Dialer) (proxy.Dialer, error) {
		v, ok := gitSyncSSHNetworks.Load(u.Host)
		if !ok {
			return nil, errors.New("Git 작업의 SSH 연결이 만료되었습니다")
		}
		return v.(*gitSyncNetwork), nil
	})
}

func newGitSyncRemote(ctx context.Context, cfg map[string]any, allowed []string, allowPrivate bool) (*gitSyncRemote, error) {
	if e := gitSyncValidateConfig(cfg); e != nil {
		return nil, e
	}
	u, ips, e := gitSyncResolve(ctx, str(cfg, "url"), allowed, allowPrivate)
	if e != nil {
		return nil, e
	}
	port := u.Port()
	if port == "" {
		port = "443"
		if u.Scheme == "ssh" {
			port = "22"
		}
	}
	network := &gitSyncNetwork{address: net.JoinHostPort(ips[0].String(), port), deadline: time.Now().Add(2 * time.Minute)}
	if d, ok := ctx.Deadline(); ok && d.Before(network.deadline) {
		network.deadline = d
	}
	network.read.Store(gitSyncMaxPackBytes + (8 << 20))
	network.write.Store(gitSyncMaxPackBytes + (8 << 20))
	network.phase.Store(2 << 20)
	stop := make(chan struct{})
	var once sync.Once
	closeNetwork := func() { once.Do(func() { close(stop); network.close() }) }
	go func() {
		select {
		case <-ctx.Done():
			closeNetwork()
		case <-stop:
		}
	}()
	endpoint, e := transport.NewEndpoint(u.String())
	if e != nil {
		closeNetwork()
		return nil, errors.New("Git 전송 주소가 올바르지 않습니다")
	}
	remote := &gitSyncRemote{Endpoint: endpoint, Close: closeNetwork, AllowPack: func() { network.phase.Store(gitSyncMaxPackBytes + (2 << 20)) }}
	if u.Scheme == "https" {
		roots, e := x509.SystemCertPool()
		if e != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if pem := str(cfg, "ca_pem"); pem != "" && !roots.AppendCertsFromPEM([]byte(pem)) {
			closeNetwork()
			return nil, errors.New("Git CA 인증서 형식을 확인하세요")
		}
		expected := net.JoinHostPort(u.Hostname(), port)
		tr := &http.Transport{Proxy: nil, DisableCompression: true, DisableKeepAlives: true, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 20 * time.Second, MaxResponseHeaderBytes: 64 << 10, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: u.Hostname()}, DialContext: func(dialCtx context.Context, kind, address string) (net.Conn, error) {
			if address != expected {
				return nil, errors.New("Git HTTPS 호스트가 변경되었습니다")
			}
			return network.DialContext(dialCtx, kind, network.address)
		}}
		client := &http.Client{Transport: gitSyncHTTPTransport{tr}, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		remote.Transport = gitHTTP.NewClientWithOptions(client, &gitHTTP.ClientOptions{RedirectPolicy: gitHTTP.NoFollowRedirects})
		if password := str(cfg, "password"); password != "" {
			username := str(cfg, "username")
			if username == "" {
				username = "git"
			}
			remote.Auth = &gitHTTP.BasicAuth{Username: username, Password: password}
		}
		return remote, nil
	}
	key, _, _, rest, e := ssh.ParseAuthorizedKey([]byte(str(cfg, "host_key")))
	if e != nil || len(rest) > 0 || key.Type() == ssh.KeyAlgoDSA {
		closeNetwork()
		return nil, errors.New("고정 SSH 서버 공개 키를 확인하세요")
	}
	var signer ssh.Signer
	if password := str(cfg, "private_key_password"); password != "" {
		signer, e = ssh.ParsePrivateKeyWithPassphrase([]byte(str(cfg, "private_key")), []byte(password))
	} else {
		signer, e = ssh.ParsePrivateKey([]byte(str(cfg, "private_key")))
	}
	if e != nil {
		closeNetwork()
		return nil, errors.New("SSH 개인 키 또는 키 암호를 확인하세요")
	}
	auth := &gitSSH.PublicKeys{User: u.User.Username(), Signer: signer, HostKeyCallbackHelper: gitSSH.HostKeyCallbackHelper{HostKeyCallback: ssh.FixedHostKey(key)}}
	algorithms := []string{key.Type()}
	if key.Type() == ssh.KeyAlgoRSA {
		algorithms = []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256}
	}
	// go-git's override replaces the whole ClientConfig, including zero fields.
	// Supply the full explicit identity and callback, not a partial override.
	remote.Transport = gitSSH.NewClient(&ssh.ClientConfig{User: u.User.Username(), Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.FixedHostKey(key), Timeout: 15 * time.Second, HostKeyAlgorithms: algorithms})
	remote.Auth = auth
	endpoint.Host = ips[0].String()
	endpoint.Port, _ = strconv.Atoi(port)
	id := newID()
	gitSyncSSHNetworks.Store(id, network)
	endpoint.Proxy.URL = "madi-git-direct://" + id
	remote.Close = func() { gitSyncSSHNetworks.Delete(id); closeNetwork() }
	return remote, nil
}
