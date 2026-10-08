// Package mail delivers event notifications through a company SMTP relay.
//
// An internal relay usually accepts mail on port 25 with no credentials and
// no TLS, so authentication and encryption are optional and the session
// adapts to whatever the relay advertises. Nothing here blocks a request:
// sending happens in the background and every attempt is recorded so an
// administrator can see what left the building. The configuration lives in
// the shared settings table under the same mail.* keys every internal
// service uses, and the password is never handed back.
package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

var (
	ErrDisabled = errors.New("mail is disabled")
	ErrInvalid  = errors.New("invalid mail configuration")
)

// Event names. Each one has a mail.notify_* switch so an administrator can
// silence one kind without turning the relay off.
const (
	// EventAccountLocked: a local account was locked after repeated
	// failures. The owner cannot get in until somebody unlocks it, and the
	// administrators would otherwise learn about it from a phone call.
	EventAccountLocked = "account.locked"
	// EventAccountUnlocked: an administrator unlocked the account. The owner
	// has been waiting for exactly this.
	EventAccountUnlocked = "account.unlocked"
	// EventAccountCreated: an administrator created a local account. The new
	// operator is waiting to hear that they can sign in.
	EventAccountCreated = "account.created"
	// EventAgentUpdateHalted: an operator stopped an agent release rollout.
	// Everyone else watching the rollout needs to know it did not stall on
	// its own.
	EventAgentUpdateHalted = "agent_update.halted"
	EventTest              = "test"
)

// Settings keys, spelled exactly as the company standard lists them.
const (
	KeyEnabled        = "mail.enabled"
	KeyHost           = "mail.smtp_host"
	KeyPort           = "mail.smtp_port"
	KeySecurity       = "mail.security"
	KeySkipTLSVerify  = "mail.skip_tls_verify"
	KeyUsername       = "mail.username"
	KeyPassword       = "mail.password"
	KeyFromAddress    = "mail.from_address"
	KeyFromName       = "mail.from_name"
	KeyBaseURL        = "mail.base_url"
	KeyTimeoutSeconds = "mail.timeout_seconds"
	KeyPrefix         = "mail."
	notifyPrefix      = "mail.notify_"
)

// EventSettings maps each event to the switch that silences it.
var EventSettings = map[string]string{
	EventAccountLocked:     notifyPrefix + "account_locked",
	EventAccountUnlocked:   notifyPrefix + "account_unlocked",
	EventAccountCreated:    notifyPrefix + "account_created",
	EventAgentUpdateHalted: notifyPrefix + "agent_update_halted",
}

// Events lists the switchable events in the order the console shows them.
var Events = []string{
	EventAccountLocked, EventAccountUnlocked, EventAccountCreated,
	EventAgentUpdateHalted,
}

// Securities are the accepted mail.security values.
var Securities = []string{"auto", "none", "starttls", "tls"}

const (
	defaultPort     = 25
	defaultSecurity = "auto"
	defaultTimeout  = 10 * time.Second
	defaultFromName = "Invenqor"
)

type Config struct {
	Enabled     bool
	Host        string
	Port        int
	Security    string
	SkipVerify  bool
	Username    string
	Password    string
	FromAddress string
	FromName    string
	BaseURL     string
	Timeout     time.Duration
	// Events holds the explicit switches. An event that is absent is on.
	Events map[string]bool
}

// Default is what a fresh installation runs with: off, and shaped for the
// common internal relay should an administrator turn it on.
func Default() Config {
	return Config{
		Port:     defaultPort,
		Security: defaultSecurity,
		FromName: defaultFromName,
		Timeout:  defaultTimeout,
		Events:   map[string]bool{},
	}
}

// FromValues builds the configuration from the decoded mail.* settings.
// Missing keys keep their defaults, so a partially filled form still reads.
func FromValues(values map[string]any) Config {
	config := Default()
	config.Enabled = boolValue(values, KeyEnabled, false)
	config.Host = stringValue(values, KeyHost, "")
	config.Security = strings.ToLower(stringValue(values, KeySecurity, defaultSecurity))
	config.SkipVerify = boolValue(values, KeySkipTLSVerify, false)
	config.Username = stringValue(values, KeyUsername, "")
	config.Password = stringValue(values, KeyPassword, "")
	config.FromAddress = stringValue(values, KeyFromAddress, "")
	config.FromName = stringValue(values, KeyFromName, defaultFromName)
	config.BaseURL = strings.TrimRight(stringValue(values, KeyBaseURL, ""), "/")
	if port, ok := numberValue(values, KeyPort); ok && port > 0 {
		config.Port = port
	}
	if seconds, ok := numberValue(values, KeyTimeoutSeconds); ok && seconds > 0 {
		config.Timeout = time.Duration(seconds) * time.Second
	}
	// A relay on the implicit TLS port needs no extra configuration.
	if config.Security == defaultSecurity && config.Port == 465 {
		config.Security = "tls"
	}
	for event, key := range EventSettings {
		if enabled, ok := values[key].(bool); ok {
			config.Events[event] = enabled
		}
	}
	if config.FromAddress == "" && config.Host != "" {
		config.FromAddress = "invenqor@" + config.Host
	}
	return config
}

// Values is the inverse of FromValues: the settings rows to store. The
// password is deliberately absent; the caller seals it separately.
func (c Config) Values() map[string]any {
	values := map[string]any{
		KeyEnabled:        c.Enabled,
		KeyHost:           c.Host,
		KeyPort:           c.Port,
		KeySecurity:       c.Security,
		KeySkipTLSVerify:  c.SkipVerify,
		KeyUsername:       c.Username,
		KeyFromAddress:    c.FromAddress,
		KeyFromName:       c.FromName,
		KeyBaseURL:        c.BaseURL,
		KeyTimeoutSeconds: int(c.Timeout / time.Second),
	}
	for event, key := range EventSettings {
		values[key] = c.Allows(event)
	}
	return values
}

// Public is what the settings API returns. The password never appears;
// only whether one is stored.
func (c Config) Public() map[string]any {
	events := make(map[string]bool, len(Events))
	for _, event := range Events {
		events[event] = c.Allows(event)
	}
	return map[string]any{
		"enabled":             c.Enabled,
		"ready":               c.Enabled && c.Validate() == nil,
		"smtp_host":           c.Host,
		"smtp_port":           c.Port,
		"security":            c.Security,
		"securities":          Securities,
		"skip_tls_verify":     c.SkipVerify,
		"username":            c.Username,
		"password_configured": c.Password != "",
		"from_address":        c.FromAddress,
		"from_name":           c.FromName,
		"base_url":            c.BaseURL,
		"timeout_seconds":     int(c.Timeout / time.Second),
		"events":              events,
		"event_names":         Events,
	}
}

// Normalize trims the free-text fields and fills the defaults a stored value
// written by an older version may lack.
func (c Config) Normalize() Config {
	c.Host = strings.TrimSpace(c.Host)
	c.Security = strings.ToLower(strings.TrimSpace(c.Security))
	if c.Security == "" {
		c.Security = defaultSecurity
	}
	c.Username = strings.TrimSpace(c.Username)
	c.FromAddress = strings.TrimSpace(c.FromAddress)
	c.FromName = strings.TrimSpace(c.FromName)
	if c.FromName == "" {
		c.FromName = defaultFromName
	}
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.Port == 0 {
		c.Port = defaultPort
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	if c.Events == nil {
		c.Events = map[string]bool{}
	}
	return c
}

// Validate reports what is wrong with the configuration. The value shape is
// checked even while mail is off, so a port or security mode that could never
// work is refused the moment it is typed; the relay itself is only required
// once mail is on.
func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("%w: %s must be between 1 and 65535", ErrInvalid, KeyPort)
	}
	known := false
	for _, security := range Securities {
		if c.Security == security {
			known = true
		}
	}
	if !known {
		return fmt.Errorf("%w: %s must be one of %s", ErrInvalid, KeySecurity, strings.Join(Securities, ", "))
	}
	if c.FromAddress != "" && !isAddress(c.FromAddress) {
		return fmt.Errorf("%w: %s must be an email address", ErrInvalid, KeyFromAddress)
	}
	if c.BaseURL != "" && !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("%w: %s must be an http(s) address", ErrInvalid, KeyBaseURL)
	}
	if !c.Enabled {
		return nil
	}
	if c.Host == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalid, KeyHost)
	}
	if c.FromAddress == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalid, KeyFromAddress)
	}
	return nil
}

// Allows reports whether an event should be delivered. Unknown events are
// sent, so adding a notification never requires a settings change first.
func (c Config) Allows(event string) bool {
	if enabled, known := c.Events[event]; known {
		return enabled
	}
	return true
}

// Address is the RFC 5322 From header value.
func (c Config) Address() string {
	if c.FromName != "" {
		return fmt.Sprintf("%s <%s>", c.FromName, c.FromAddress)
	}
	return c.FromAddress
}

func (c Config) endpoint() string { return net.JoinHostPort(c.Host, fmt.Sprint(c.Port)) }

func (c Config) tlsConfig() *tls.Config {
	return &tls.Config{
		ServerName:         c.Host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.SkipVerify, //nolint:gosec // opt-in for internal relays with private certificates
	}
}

// IsAddress reports whether a recipient looks like a mailbox. It is
// deliberately loose: the relay is the authority on what it accepts.
func IsAddress(value string) bool { return isAddress(value) }

func isAddress(value string) bool {
	at := strings.LastIndex(value, "@")
	return at > 0 && at < len(value)-1 &&
		!strings.ContainsAny(value, " \t\r\n<>,;\"")
}

type Message struct {
	To      string
	Subject string
	Body    string
}

// Deliver opens a connection and sends one message. Every error names the
// SMTP step that failed, which is what the test button shows the operator.
// The password never appears in an error.
func Deliver(ctx context.Context, config Config, message Message) error {
	if err := config.Validate(); err != nil {
		return err
	}
	if config.Host == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalid, KeyHost)
	}
	if !isAddress(message.To) {
		return fmt.Errorf("%w: recipient is required", ErrInvalid)
	}
	client, err := dial(ctx, config)
	if err != nil {
		return err
	}
	defer func() { _ = client.Close() }()
	if err := startSession(client, config); err != nil {
		return err
	}
	if err := client.Mail(config.FromAddress); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(message.To); err != nil {
		return fmt.Errorf("RCPT TO failed: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}
	if _, err := writer.Write([]byte(compose(config, message, time.Now()))); err != nil {
		return fmt.Errorf("message body failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("message end failed: %w", err)
	}
	return client.Quit()
}

func dial(ctx context.Context, config Config) (*smtp.Client, error) {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	dialer := &net.Dialer{Timeout: timeout}
	var connection net.Conn
	var err error
	if config.Security == "tls" {
		tlsDialer := &tls.Dialer{NetDialer: dialer, Config: config.tlsConfig()}
		connection, err = tlsDialer.DialContext(ctx, "tcp", config.endpoint())
	} else {
		connection, err = dialer.DialContext(ctx, "tcp", config.endpoint())
	}
	if err != nil {
		return nil, fmt.Errorf("SMTP connect to %s failed: %w", config.endpoint(), err)
	}
	// The relay must answer every step within the timeout too; a relay that
	// accepts the connection and then hangs is the common failure.
	deadline := time.Now().Add(timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = connection.SetDeadline(deadline)
	client, err := smtp.NewClient(connection, config.Host)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("SMTP greeting failed: %w", err)
	}
	return client, nil
}

// startSession upgrades and authenticates only as far as the relay allows,
// so an unauthenticated internal relay works with the same settings as a
// hosted provider that demands both.
func startSession(client *smtp.Client, config Config) error {
	if err := client.Hello(helloName(config)); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}
	if config.Security == "starttls" || config.Security == "auto" {
		if supported, _ := client.Extension("STARTTLS"); supported {
			if err := client.StartTLS(config.tlsConfig()); err != nil {
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		} else if config.Security == "starttls" {
			return fmt.Errorf("%w: the relay does not offer STARTTLS", ErrInvalid)
		}
	}
	if config.Username == "" {
		return nil
	}
	supported, mechanisms := client.Extension("AUTH")
	if !supported {
		return fmt.Errorf("%w: the relay does not offer AUTH; clear %s to send without credentials", ErrInvalid, KeyUsername)
	}
	var auth smtp.Auth
	switch {
	case strings.Contains(strings.ToUpper(mechanisms), "PLAIN"):
		auth = smtp.PlainAuth("", config.Username, config.Password, config.Host)
	case strings.Contains(strings.ToUpper(mechanisms), "LOGIN"):
		auth = loginAuth{username: config.Username, password: config.Password, host: config.Host}
	default:
		auth = smtp.CRAMMD5Auth(config.Username, config.Password)
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("AUTH failed: %w", err)
	}
	return nil
}

// helloName keeps the EHLO name to the sender domain, which relays that
// check the greeting accept more readily than a container hostname.
func helloName(config Config) string {
	if index := strings.LastIndex(config.FromAddress, "@"); index >= 0 && index+1 < len(config.FromAddress) {
		return config.FromAddress[index+1:]
	}
	return "localhost"
}

// loginAuth implements the LOGIN mechanism several corporate relays use
// instead of PLAIN. The standard library ships only PLAIN and CRAM-MD5.
type loginAuth struct{ username, password, host string }

func (a loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	if !server.TLS && server.Name != a.host {
		return "", nil, errors.New("LOGIN authentication requires a trusted relay")
	}
	return "LOGIN", nil, nil
}

func (a loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimRight(string(fromServer), ": ")) {
	case "username":
		return []byte(a.username), nil
	case "password":
		return []byte(a.password), nil
	}
	return nil, fmt.Errorf("unexpected LOGIN challenge %q", fromServer)
}

func boolValue(values map[string]any, key string, fallback bool) bool {
	if value, ok := values[key].(bool); ok {
		return value
	}
	return fallback
}

func stringValue(values map[string]any, key, fallback string) string {
	if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func numberValue(values map[string]any, key string) (int, bool) {
	switch typed := values[key].(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	}
	return 0, false
}
