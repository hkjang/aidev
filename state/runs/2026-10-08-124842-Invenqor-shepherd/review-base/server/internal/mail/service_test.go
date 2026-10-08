package mail

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

type memoryDirectory map[string]string

func (d memoryDirectory) LookupEmails(_ context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	for _, id := range ids {
		if address, ok := d[id]; ok {
			out[id] = address
		}
	}
	return out, nil
}

type recordingSender struct {
	mu   sync.Mutex
	sent []Message
	fail func(Message) error
}

func (r *recordingSender) send(_ context.Context, _ Config, message Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail != nil {
		if err := r.fail(message); err != nil {
			return err
		}
	}
	r.sent = append(r.sent, message)
	return nil
}

func (r *recordingSender) recipients() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, message := range r.sent {
		out = append(out, message.To)
	}
	return out
}

func newTestService(t *testing.T, config Config) (*Service, *recordingSender) {
	t.Helper()
	runtime := storagetest.Open(t)
	t.Cleanup(func() { _ = runtime.Close() })
	sender := &recordingSender{}
	directory := memoryDirectory{
		"alice": "alice@corp.example", "bob": "bob@corp.example", "carol": "carol@corp.example",
	}
	service := NewService(runtime.DB(), func(context.Context) (Config, error) { return config, nil },
		directory, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service.SetSender(sender.send)
	return service, sender
}

func enabledConfig() Config {
	config := Default()
	config.Enabled = true
	config.Host = "relay.corp.example"
	config.FromAddress = "invenqor@corp.example"
	return config
}

func TestNothingIsSentWhileMailIsOff(t *testing.T) {
	service, sender := newTestService(t, Default())
	service.Notify(context.Background(), AccountUnlocked("alice", "", "Admin"), "bob", []string{"alice"})
	service.Wait()
	if len(sender.recipients()) != 0 {
		t.Fatalf("sent %v while off", sender.recipients())
	}
	page, err := service.Deliveries(context.Background(), "", 50)
	if err != nil || page.Summary.Total != 0 {
		t.Fatalf("deliveries = %+v, err = %v; nothing must be recorded while off", page, err)
	}
	if err := service.SendNow(context.Background(), TestMessage(), "bob", "bob@corp.example"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("SendNow() while off error = %v", err)
	}
}

// Enabled but without a relay address: nothing goes out and nothing is
// recorded, because there is nowhere to record an attempt against.
func TestAnIncompleteConfigurationSendsNothing(t *testing.T) {
	config := Default()
	config.Enabled = true
	service, sender := newTestService(t, config)
	service.Notify(context.Background(), AccountUnlocked("alice", "", "Admin"), "bob", []string{"alice"})
	service.Wait()
	if len(sender.recipients()) != 0 {
		t.Fatalf("sent %v without a relay", sender.recipients())
	}
	if err := service.SendNow(context.Background(), TestMessage(), "bob", "bob@corp.example"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("SendNow() error = %v, want the missing host named", err)
	}
}

func TestTheActorIsNeverToldAboutTheirOwnAction(t *testing.T) {
	service, sender := newTestService(t, enabledConfig())
	service.Notify(context.Background(), AgentUpdateHalted("1.2.3", "x86_64", "Bob", ""),
		"bob", []string{"alice", "bob", "carol", "alice", "nobody", "direct@corp.example"})
	service.Wait()
	got := sender.recipients()
	want := map[string]bool{"alice@corp.example": true, "carol@corp.example": true, "direct@corp.example": true}
	if len(got) != len(want) {
		t.Fatalf("recipients = %v, want %v", got, want)
	}
	for _, address := range got {
		if !want[address] {
			t.Fatalf("unexpected recipient %s in %v", address, got)
		}
	}
}

func TestAnEventSwitchStopsOnlyThatEvent(t *testing.T) {
	config := enabledConfig()
	config.Events[EventAccountCreated] = false
	service, sender := newTestService(t, config)
	service.Notify(context.Background(), AccountCreated("alice", "", "Admin", nil), "bob", []string{"alice"})
	service.Notify(context.Background(), AccountUnlocked("alice", "", "Admin"), "bob", []string{"alice"})
	service.Wait()
	if got := sender.recipients(); len(got) != 1 {
		t.Fatalf("recipients = %v, want only the unlock mail", got)
	}
	page, err := service.Deliveries(context.Background(), "", 50)
	if err != nil || len(page.Items) != 1 || page.Items[0].Event != EventAccountUnlocked {
		t.Fatalf("deliveries = %+v, err = %v", page, err)
	}
}

// Every attempt is in the log - the ones that arrived and the ones that did
// not - with the subject but never the body.
func TestSuccessAndFailureAreBothRecorded(t *testing.T) {
	service, sender := newTestService(t, enabledConfig())
	sender.fail = func(message Message) error {
		if message.To == "carol@corp.example" {
			return errors.New("SMTP connect to relay.corp.example:25 failed: connection refused")
		}
		return nil
	}
	notification := AccountLocked("alice", "Alice", "10.0.0.9", time.Time{})
	service.Notify(context.Background(), notification, "", []string{"alice", "carol"})
	service.Wait()
	page, err := service.Deliveries(context.Background(), "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if page.Summary.Total != 2 || page.Summary.Status[StatusSent] != 1 || page.Summary.Status[StatusFailed] != 1 {
		t.Fatalf("summary = %+v", page.Summary)
	}
	for _, item := range page.Items {
		if item.Subject != notification.Subject || item.Event != EventAccountLocked {
			t.Fatalf("delivery = %+v", item)
		}
		switch item.Recipient {
		case "alice@corp.example":
			if item.Status != StatusSent || item.Attempts != 1 || item.ErrorMessage != "" {
				t.Fatalf("sent delivery = %+v", item)
			}
		case "carol@corp.example":
			if item.Status != StatusFailed || item.Attempts != 2 || item.ErrorMessage == "" {
				t.Fatalf("failed delivery = %+v", item)
			}
		default:
			t.Fatalf("unexpected recipient %+v", item)
		}
		if !item.CreatedAt.Valid || !item.UpdatedAt.Valid {
			t.Fatalf("timestamps missing: %+v", item)
		}
	}
	failed, err := service.Deliveries(context.Background(), StatusFailed, 50)
	if err != nil || len(failed.Items) != 1 || failed.Summary.Total != 2 {
		t.Fatalf("failed filter = %+v, err = %v", failed, err)
	}
	var bodies int
	if err := service.db.QueryRow(`SELECT COUNT(*) FROM mail_deliveries WHERE subject LIKE '%마지막 시도 출처%'`).Scan(&bodies); err != nil || bodies != 0 {
		t.Fatalf("body text found in the delivery log: %d, %v", bodies, err)
	}
}
