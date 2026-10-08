package mail

import (
	"fmt"
	"mime"
	"strings"
	"time"
)

// compose builds a plain-text MIME message. Korean subjects are encoded so
// relays and clients that predate UTF-8 headers still show them.
func compose(config Config, message Message, now time.Time) string {
	var builder strings.Builder
	builder.WriteString("From: " + encodeAddress(config.Address()) + "\r\n")
	builder.WriteString("To: " + message.To + "\r\n")
	builder.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", message.Subject) + "\r\n")
	builder.WriteString("Date: " + now.Format(time.RFC1123Z) + "\r\n")
	builder.WriteString("MIME-Version: 1.0\r\n")
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	builder.WriteString("Auto-Submitted: auto-generated\r\n")
	builder.WriteString("X-Invenqor-Notification: 1\r\n")
	builder.WriteString("\r\n")
	builder.WriteString(normalizeBody(message.Body))
	return builder.String()
}

func encodeAddress(address string) string {
	open := strings.LastIndex(address, "<")
	if open <= 0 {
		return address
	}
	return mime.QEncoding.Encode("utf-8", strings.TrimSpace(address[:open])) + " " + address[open:]
}

// normalizeBody uses CRLF line endings. Dot-stuffing is the SMTP client's
// job: the DATA writer escapes a leading dot itself.
func normalizeBody(body string) string {
	body = strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n")
	if !strings.HasSuffix(body, "\r\n") {
		body += "\r\n"
	}
	return body
}

// Notification is the content of one event mail before recipients are
// resolved. Link is a console path; Render makes it absolute with
// mail.base_url and drops it when no base URL is configured.
type Notification struct {
	Event   string
	Subject string
	Lines   []string
	Link    string
}

// Render turns a notification into the message body, appending the console
// link and a footer that says why the mail arrived.
func (n Notification) Render(config Config) string {
	lines := append([]string{}, n.Lines...)
	if link := n.absoluteLink(config); link != "" {
		lines = append(lines, "", "바로 열기: "+link)
	}
	lines = append(lines, "", "—",
		"이 메일은 Invenqor 관리 콘솔의 알림 설정에 따라 자동으로 발송되었습니다.")
	return strings.Join(lines, "\n")
}

func (n Notification) absoluteLink(config Config) string {
	if n.Link == "" {
		return ""
	}
	if strings.HasPrefix(n.Link, "http://") || strings.HasPrefix(n.Link, "https://") {
		return n.Link
	}
	if config.BaseURL == "" {
		return ""
	}
	return config.BaseURL + "/" + strings.TrimLeft(n.Link, "/")
}

// AccountLocked tells the owner and the administrators that repeated failed
// logins locked an account. The failing source is named so an owner who did
// not type anything knows somebody else did.
func AccountLocked(username, displayName, sourceIP string, until time.Time) Notification {
	who := label(username, displayName)
	lines := []string{
		fmt.Sprintf("%s 계정이 로그인 실패 누적으로 잠겼습니다.", who),
	}
	if strings.TrimSpace(sourceIP) != "" {
		lines = append(lines, fmt.Sprintf("마지막 시도 출처: %s", strings.TrimSpace(sourceIP)))
	}
	if !until.IsZero() {
		lines = append(lines, fmt.Sprintf("자동 해제 예정: %s", until.UTC().Format("2006-01-02 15:04 UTC")))
	}
	lines = append(lines, "",
		"본인이 시도한 것이 아니라면 관리자에게 알리십시오.",
		"관리자는 사용자 관리에서 바로 잠금을 해제할 수 있습니다.")
	return Notification{
		Event:   EventAccountLocked,
		Subject: fmt.Sprintf("[Invenqor] %s 계정이 잠겼습니다", who),
		Lines:   lines,
		Link:    "/#/users",
	}
}

// AccountUnlocked tells the owner they can sign in again.
func AccountUnlocked(username, displayName, actor string) Notification {
	who := label(username, displayName)
	return Notification{
		Event:   EventAccountUnlocked,
		Subject: fmt.Sprintf("[Invenqor] %s 계정 잠금이 해제되었습니다", who),
		Lines: []string{
			fmt.Sprintf("관리자 %s 님이 %s 계정의 잠금을 해제했습니다.", actor, who),
			"지금 다시 로그인할 수 있습니다.",
		},
		Link: "/",
	}
}

// AccountCreated tells a new operator that an account is waiting for them.
// The password is never in the mail; the administrator hands it over
// separately.
func AccountCreated(username, displayName, actor string, roles []string) Notification {
	lines := []string{
		fmt.Sprintf("관리자 %s 님이 회원님의 Invenqor 계정을 만들었습니다.", actor),
		fmt.Sprintf("사용자 이름: %s", username),
	}
	if len(roles) > 0 {
		lines = append(lines, fmt.Sprintf("역할: %s", strings.Join(roles, ", ")))
	}
	lines = append(lines, "",
		"초기 비밀번호는 이 메일에 담지 않습니다. 관리자에게 직접 받으십시오.",
		"첫 로그인 뒤 계정 화면에서 비밀번호를 바꾸십시오.")
	return Notification{
		Event:   EventAccountCreated,
		Subject: fmt.Sprintf("[Invenqor] %s 계정이 준비되었습니다", label(username, displayName)),
		Lines:   lines,
		Link:    "/",
	}
}

// AgentUpdateHalted tells the other operators that a rollout was stopped on
// purpose, so a rollout that stays put is not mistaken for a stalled one.
func AgentUpdateHalted(version, architecture, actor, reason string) Notification {
	lines := []string{
		fmt.Sprintf("%s 님이 Agent %s (%s) 배포를 0%% 로 멈췄습니다.", actor, version, architecture),
		"이미 받은 Agent 는 그대로이고, 아직 받지 않은 Agent 는 이 버전을 받지 않습니다.",
	}
	if strings.TrimSpace(reason) != "" {
		lines = append(lines, "", "사유: "+strings.TrimSpace(reason))
	}
	return Notification{
		Event:   EventAgentUpdateHalted,
		Subject: fmt.Sprintf("[Invenqor] Agent %s 배포가 중단되었습니다", version),
		Lines:   lines,
		Link:    "/#/agents",
	}
}

// TestMessage proves the relay works from the settings screen.
func TestMessage() Notification {
	return Notification{
		Event:   EventTest,
		Subject: "[Invenqor] SMTP 시험 발송",
		Lines: []string{
			"Invenqor 관리 콘솔에서 보낸 시험 메일입니다.",
			"이 메일을 받았다면 SMTP 릴레이 설정이 정상입니다.",
		},
	}
}

func label(username, displayName string) string {
	if name := strings.TrimSpace(displayName); name != "" && name != username {
		return fmt.Sprintf("%s (%s)", name, username)
	}
	return username
}
