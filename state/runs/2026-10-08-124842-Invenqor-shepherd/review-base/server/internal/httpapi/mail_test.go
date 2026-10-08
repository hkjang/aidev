package httpapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/mail"
	"github.com/hkjang/invenqor/server/internal/storagetest"
)

type capturedMail struct {
	mu       sync.Mutex
	messages []mail.Message
}

func (c *capturedMail) send(_ context.Context, _ mail.Config, message mail.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messages = append(c.messages, message)
	return nil
}

func (c *capturedMail) drain() []mail.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.messages
	c.messages = nil
	return out
}

func getMailSettings(t *testing.T, server *Server, cookie *http.Cookie) map[string]any {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/mail", nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET mail settings status = %d body = %s", response.Code, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func patchMail(t *testing.T, server *Server, cookie *http.Cookie, csrf string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	return performAuthenticatedJSON(t, server, http.MethodPatch, "/api/v1/admin/settings/mail", body, cookie, csrf)
}

func mailDeliveries(t *testing.T, server *Server, cookie *http.Cookie, status string) mail.Page {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/mail/deliveries?status="+status, nil)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET deliveries status = %d body = %s", response.Code, response.Body.String())
	}
	var page mail.Page
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	return page
}

// A fresh installation: mail is off, nothing is stored, and the settings
// API says so without writing a row.
func TestMailIsOffByDefaultAndThePasswordNeverComesBack(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	settings := getMailSettings(t, server, cookie)
	if settings["enabled"] != false || settings["smtp_port"] != float64(25) ||
		settings["security"] != "auto" || settings["password_configured"] != false {
		t.Fatalf("default mail settings = %v", settings)
	}
	var rows int
	if err := runtime.DB().QueryRow(`SELECT COUNT(*) FROM settings WHERE key LIKE 'mail.%'`).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("mail rows after a read = %d, err = %v", rows, err)
	}

	saved := patchMail(t, server, cookie, csrf, map[string]any{
		"smtp_host": "relay.corp.example", "username": "relay-user",
		"password": "hunter2-secret", "from_address": "invenqor@corp.example",
		"reason": "relay pilot",
	})
	if saved.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d body = %s", saved.Code, saved.Body.String())
	}
	if strings.Contains(saved.Body.String(), "hunter2") {
		t.Fatalf("PATCH response returns the password: %s", saved.Body.String())
	}
	settings = getMailSettings(t, server, cookie)
	if settings["password_configured"] != true || settings["username"] != "relay-user" || settings["enabled"] != false {
		t.Fatalf("saved settings = %v", settings)
	}
	if _, present := settings["password"]; present {
		t.Fatalf("GET returns a password field: %v", settings)
	}

	// The stored row is sealed, listed as masked, and refused by the generic
	// settings endpoint so a plaintext copy can never be written.
	var stored string
	if err := runtime.DB().QueryRow(`SELECT value_json FROM settings WHERE key=$1 AND secret=TRUE`, mail.KeyPassword).Scan(&stored); err != nil {
		t.Fatalf("password row: %v", err)
	}
	if strings.Contains(stored, "hunter2") {
		t.Fatalf("password stored in clear: %s", stored)
	}
	list := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings", nil)
	list.AddCookie(cookie)
	listed := httptest.NewRecorder()
	server.Handler().ServeHTTP(listed, list)
	if listed.Code != http.StatusOK || strings.Contains(listed.Body.String(), "hunter2") ||
		!strings.Contains(listed.Body.String(), `"key":"mail.password","pending":false,"secret":true`) ||
		!strings.Contains(listed.Body.String(), `"value":{"configured":true,"masked":true}`) {
		t.Fatalf("generic list = %d %s", listed.Code, listed.Body.String())
	}
	generic := performAuthenticatedJSON(t, server, http.MethodPatch, "/api/v1/admin/settings",
		map[string]any{"settings": []map[string]any{{"key": mail.KeyPassword, "value": `"plain"`}}}, cookie, csrf)
	if generic.Code != http.StatusConflict {
		t.Fatalf("generic write of mail.password status = %d body = %s", generic.Code, generic.Body.String())
	}

	// Saving again without a password keeps it; an empty one clears it.
	patchMail(t, server, cookie, csrf, map[string]any{"from_name": "Invenqor Ops"})
	if getMailSettings(t, server, cookie)["password_configured"] != true {
		t.Fatal("a save without a password field must keep the stored one")
	}
	patchMail(t, server, cookie, csrf, map[string]any{"password": ""})
	if getMailSettings(t, server, cookie)["password_configured"] != false {
		t.Fatal("an empty password must clear the stored one")
	}

	// A configuration that could never work is refused with the reason.
	refused := patchMail(t, server, cookie, csrf, map[string]any{"security": "ssl"})
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "mail.security") {
		t.Fatalf("invalid security status = %d body = %s", refused.Code, refused.Body.String())
	}
	refused = patchMail(t, server, cookie, csrf, map[string]any{"enabled": true, "smtp_host": ""})
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "mail.smtp_host") {
		t.Fatalf("enable without host status = %d body = %s", refused.Code, refused.Body.String())
	}
	if getMailSettings(t, server, cookie)["enabled"] != false {
		t.Fatal("a refused save must not change anything")
	}
}

// The events people wait for go out in the background, never to the person
// who caused them, and each is recorded. A dead relay fails the delivery,
// not the request.
func TestAccountEventsAreMailedToThePeopleWaiting(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	if _, err := runtime.DB().Exec(`UPDATE users SET email='admin@corp.example' WHERE username='admin.user'`); err != nil {
		t.Fatal(err)
	}
	captured := &capturedMail{}
	server.mail.SetSender(captured.send)

	// Off: creating a user sends nothing and records nothing.
	create := performAuthenticatedJSON(t, server, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"username": "quiet.user", "display_name": "Quiet", "email": "quiet@corp.example",
		"password": "QuietUser!42", "roles": []string{"viewer"},
	}, cookie, csrf)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", create.Code, create.Body.String())
	}
	server.mail.Wait()
	if got := captured.drain(); len(got) != 0 {
		t.Fatalf("mail while off: %+v", got)
	}

	if saved := patchMail(t, server, cookie, csrf, map[string]any{
		"enabled": true, "smtp_host": "relay.corp.example", "from_address": "invenqor@corp.example",
		"base_url": "https://invenqor.corp.example",
	}); saved.Code != http.StatusOK {
		t.Fatalf("enable status = %d body = %s", saved.Code, saved.Body.String())
	}

	// account.created: the new operator hears about it, the administrator
	// who created them does not.
	create = performAuthenticatedJSON(t, server, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"username": "new.user", "display_name": "New Operator", "email": "new@corp.example",
		"password": "NewOperator!42", "roles": []string{"viewer"},
	}, cookie, csrf)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", create.Code, create.Body.String())
	}
	var created struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	server.mail.Wait()
	got := captured.drain()
	if len(got) != 1 || got[0].To != "new@corp.example" || !strings.Contains(got[0].Subject, "계정이 준비") {
		t.Fatalf("account.created mail = %+v", got)
	}
	if strings.Contains(got[0].Body, "NewOperator!42") {
		t.Fatalf("the password is in the mail:\n%s", got[0].Body)
	}
	if !strings.Contains(got[0].Body, "바로 열기: https://invenqor.corp.example/") {
		t.Fatalf("mail lacks the console link:\n%s", got[0].Body)
	}

	// account.locked: five wrong passwords lock the account; the owner and
	// the administrator are told, once each.
	for range 5 {
		performJSON(t, server, http.MethodPost, "/api/v1/auth/local/login",
			map[string]string{"username": "new.user", "password": "wrong-password"}, nil)
	}
	server.mail.Wait()
	got = captured.drain()
	recipients := map[string]bool{}
	for _, message := range got {
		recipients[message.To] = true
		if !strings.Contains(message.Subject, "계정이 잠겼습니다") {
			t.Fatalf("locked subject = %q", message.Subject)
		}
	}
	if len(got) != 2 || !recipients["new@corp.example"] || !recipients["admin@corp.example"] {
		t.Fatalf("account.locked recipients = %v", recipients)
	}

	// account.unlocked: only the owner, not the administrator who did it.
	unlock := performAuthenticatedJSON(t, server, http.MethodPost,
		"/api/v1/admin/users/"+created.User.ID+"/unlock", map[string]any{}, cookie, csrf)
	if unlock.Code != http.StatusOK {
		t.Fatalf("unlock status = %d body = %s", unlock.Code, unlock.Body.String())
	}
	server.mail.Wait()
	got = captured.drain()
	if len(got) != 1 || got[0].To != "new@corp.example" || !strings.Contains(got[0].Body, "Administrator 님이") {
		t.Fatalf("account.unlocked mail = %+v", got)
	}

	// The switch for one event silences that event only.
	if saved := patchMail(t, server, cookie, csrf, map[string]any{
		"events": map[string]bool{mail.EventAccountUnlocked: false},
	}); saved.Code != http.StatusOK {
		t.Fatalf("switch status = %d body = %s", saved.Code, saved.Body.String())
	}
	performAuthenticatedJSON(t, server, http.MethodPost, "/api/v1/admin/users/"+created.User.ID+"/unlock", map[string]any{}, cookie, csrf)
	server.mail.Wait()
	if got := captured.drain(); len(got) != 0 {
		t.Fatalf("unlock mail while switched off: %+v", got)
	}
	if refused := patchMail(t, server, cookie, csrf, map[string]any{
		"events": map[string]bool{"comment.created": false},
	}); refused.Code != http.StatusBadRequest {
		t.Fatalf("unknown event status = %d", refused.Code)
	}

	page := mailDeliveries(t, server, cookie, "")
	if page.Summary.Total != 4 || page.Summary.Status[mail.StatusSent] != 4 {
		t.Fatalf("delivery summary = %+v", page.Summary)
	}

	// A relay that is down: the request still succeeds at once and the
	// attempt is recorded as failed with the reason.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	deadPort := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	server.mail.SetSender(mail.Deliver)
	if saved := patchMail(t, server, cookie, csrf, map[string]any{
		"smtp_host": "127.0.0.1", "smtp_port": deadPort, "timeout_seconds": 1,
		"events": map[string]bool{mail.EventAccountUnlocked: true},
	}); saved.Code != http.StatusOK {
		t.Fatalf("dead relay settings status = %d body = %s", saved.Code, saved.Body.String())
	}
	started := time.Now()
	unlock = performAuthenticatedJSON(t, server, http.MethodPost,
		"/api/v1/admin/users/"+created.User.ID+"/unlock", map[string]any{}, cookie, csrf)
	if unlock.Code != http.StatusOK || time.Since(started) > 2*time.Second {
		t.Fatalf("unlock with a dead relay = %d after %s", unlock.Code, time.Since(started))
	}
	server.mail.Wait()
	failed := mailDeliveries(t, server, cookie, mail.StatusFailed)
	if len(failed.Items) != 1 || failed.Items[0].Attempts != 2 ||
		!strings.Contains(failed.Items[0].ErrorMessage, "SMTP connect") {
		t.Fatalf("failed deliveries = %+v", failed.Items)
	}

	// The test button reports the same failure to the administrator.
	test := performAuthenticatedJSON(t, server, http.MethodPost, "/api/v1/admin/settings/mail/test",
		map[string]any{"recipient": "admin@corp.example"}, cookie, csrf)
	if test.Code != http.StatusBadGateway || !strings.Contains(test.Body.String(), `"sent":false`) {
		t.Fatalf("test send status = %d body = %s", test.Code, test.Body.String())
	}
	server.mail.SetSender(captured.send)
	test = performAuthenticatedJSON(t, server, http.MethodPost, "/api/v1/admin/settings/mail/test",
		map[string]any{}, cookie, csrf)
	if test.Code != http.StatusOK || !strings.Contains(test.Body.String(), `"recipient":"admin@corp.example"`) {
		t.Fatalf("test send to own address status = %d body = %s", test.Code, test.Body.String())
	}
	if got := captured.drain(); len(got) != 1 || got[0].To != "admin@corp.example" {
		t.Fatalf("test mail = %+v", got)
	}
}
