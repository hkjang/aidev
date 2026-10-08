package mail

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRelay is the smallest SMTP server that can take one message: no
// STARTTLS, no AUTH, port 25 semantics on a loopback port.
type fakeRelay struct {
	listener net.Listener
	mu       sync.Mutex
	messages []string
	rejectTo string
}

func newFakeRelay(t *testing.T) *fakeRelay {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	relay := &fakeRelay{listener: listener}
	go relay.serve()
	t.Cleanup(func() { _ = listener.Close() })
	return relay
}

func (r *fakeRelay) hostPort() (string, int) {
	address := r.listener.Addr().(*net.TCPAddr)
	return address.IP.String(), address.Port
}

func (r *fakeRelay) serve() {
	for {
		connection, err := r.listener.Accept()
		if err != nil {
			return
		}
		go r.handle(connection)
	}
}

func (r *fakeRelay) handle(connection net.Conn) {
	defer connection.Close()
	reader := bufio.NewReader(connection)
	write := func(line string) { _, _ = io.WriteString(connection, line+"\r\n") }
	write("220 relay.test ESMTP")
	var data strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			write("250-relay.test")
			write("250 SIZE 10240000")
		case strings.HasPrefix(upper, "HELO"):
			write("250 relay.test")
		case strings.HasPrefix(upper, "MAIL FROM"):
			write("250 OK")
		case strings.HasPrefix(upper, "RCPT TO"):
			if r.rejectTo != "" && strings.Contains(line, r.rejectTo) {
				write("550 5.1.1 no such user")
				continue
			}
			write("250 OK")
		case upper == "DATA":
			write("354 go ahead")
			data.Reset()
			for {
				body, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if body == ".\r\n" {
					break
				}
				// Undo the client's dot-stuffing like a real relay does.
				data.WriteString(strings.TrimPrefix(body, "."))
			}
			r.mu.Lock()
			r.messages = append(r.messages, data.String())
			r.mu.Unlock()
			write("250 queued")
		case upper == "QUIT":
			write("221 bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (r *fakeRelay) received() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.messages...)
}

func relayConfig(relay *fakeRelay) Config {
	host, port := relay.hostPort()
	config := Default()
	config.Enabled = true
	config.Host = host
	config.Port = port
	config.FromAddress = "invenqor@corp.example"
	config.FromName = "인벤코어"
	config.Timeout = 3 * time.Second
	return config
}

// The common internal relay: port 25, no credentials, no TLS. One message
// goes through with the headers a mail client needs, and a line that starts
// with a dot survives.
func TestDeliverThroughAnUnauthenticatedRelay(t *testing.T) {
	relay := newFakeRelay(t)
	config := relayConfig(relay)
	err := Deliver(context.Background(), config, Message{
		To: "ops@corp.example", Subject: "[Invenqor] 계정이 잠겼습니다",
		Body: "첫 줄\n.끝에 점\n",
	})
	if err != nil {
		t.Fatalf("Deliver() error = %v", err)
	}
	messages := relay.received()
	if len(messages) != 1 {
		t.Fatalf("relay received %d messages, want 1", len(messages))
	}
	raw := messages[0]
	for _, want := range []string{
		"From: =?utf-8?q?=EC=9D=B8=EB=B2=A4=EC=BD=94=EC=96=B4?= <invenqor@corp.example>\r\n",
		"To: ops@corp.example\r\n",
		"Subject: =?utf-8?q?",
		"Auto-Submitted: auto-generated\r\n",
		"Content-Type: text/plain; charset=UTF-8\r\n",
		"\r\n첫 줄\r\n.끝에 점\r\n",
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("message lacks %q:\n%s", want, raw)
		}
	}
}

func TestDeliverReportsTheRefusedStep(t *testing.T) {
	relay := newFakeRelay(t)
	relay.rejectTo = "nobody@"
	err := Deliver(context.Background(), relayConfig(relay), Message{To: "nobody@corp.example", Subject: "x", Body: "y"})
	if err == nil || !strings.Contains(err.Error(), "RCPT TO") {
		t.Fatalf("Deliver() error = %v, want the RCPT TO step named", err)
	}
}

// A relay that is not listening fails within the timeout and says so; the
// password is never part of the message.
func TestDeliverFailsFastWhenTheRelayIsDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().(*net.TCPAddr)
	_ = listener.Close()
	config := Default()
	config.Enabled = true
	config.Host, config.Port = address.IP.String(), address.Port
	config.FromAddress = "invenqor@corp.example"
	config.Username, config.Password = "relay-user", "hunter2-secret"
	config.Timeout = 2 * time.Second
	started := time.Now()
	err = Deliver(context.Background(), config, Message{To: "ops@corp.example", Subject: "x", Body: "y"})
	if err == nil {
		t.Fatal("Deliver() succeeded against a closed port")
	}
	if time.Since(started) > 5*time.Second {
		t.Fatalf("Deliver() took %s against a closed port", time.Since(started))
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Fatalf("error leaks the password: %v", err)
	}
}

func TestDefaultsMatchTheInternalRelay(t *testing.T) {
	config := FromValues(nil)
	if config.Enabled || config.Port != 25 || config.Security != "auto" || config.Timeout != 10*time.Second {
		t.Fatalf("defaults = %+v", config)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("an empty, disabled configuration must validate: %v", err)
	}
	config = FromValues(map[string]any{
		KeyEnabled: true, KeyHost: "relay.corp.example", KeyPort: float64(465),
		KeyTimeoutSeconds: float64(3), KeySecurity: "AUTO",
		EventSettings[EventAccountCreated]: false,
	})
	if config.Security != "tls" || config.Port != 465 || config.Timeout != 3*time.Second {
		t.Fatalf("465 should imply tls: %+v", config)
	}
	if config.FromAddress != "invenqor@relay.corp.example" {
		t.Fatalf("from address default = %q", config.FromAddress)
	}
	if config.Allows(EventAccountCreated) || !config.Allows(EventAccountLocked) || !config.Allows("unknown.event") {
		t.Fatalf("event switches = %+v", config.Events)
	}
	if !config.Public()["ready"].(bool) {
		t.Fatalf("configured relay should be ready: %+v", config.Public())
	}
	for key := range config.Public() {
		if key == "password" {
			t.Fatal("Public() must not carry the password")
		}
	}
}

func TestValidateRefusesWhatCouldNeverWork(t *testing.T) {
	enabledWithoutHost := Default()
	enabledWithoutHost.Enabled = true
	if err := enabledWithoutHost.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("enabled without host error = %v", err)
	}
	badSecurity := Default()
	badSecurity.Security = "ssl"
	if err := badSecurity.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown security error = %v", err)
	}
	badFrom := Default()
	badFrom.FromAddress = "not an address"
	if err := badFrom.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad from error = %v", err)
	}
}

func TestRenderAddsTheConsoleLinkOnlyWithABaseURL(t *testing.T) {
	notification := AccountUnlocked("jane", "Jane Doe", "Admin")
	withoutBase := notification.Render(Default())
	if strings.Contains(withoutBase, "바로 열기") {
		t.Fatalf("link without base url:\n%s", withoutBase)
	}
	config := Default()
	config.BaseURL = "https://invenqor.corp.example/"
	config = config.Normalize()
	withBase := notification.Render(config)
	if !strings.Contains(withBase, "바로 열기: https://invenqor.corp.example/") {
		t.Fatalf("link with base url:\n%s", withBase)
	}
	if !strings.Contains(notification.Subject, "Jane Doe (jane)") {
		t.Fatalf("subject = %q", notification.Subject)
	}
}
