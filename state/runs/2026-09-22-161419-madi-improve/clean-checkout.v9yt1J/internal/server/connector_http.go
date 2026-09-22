package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type connectorConfig struct {
	ID, WorkspaceID, SpaceID, OwnerID, ServiceID, Name, Kind, BaseURL string
	Config, Credentials                                               map[string]any
	Enabled                                                           bool
	Revision, Interval                                                int
	Cursor                                                            string
}
type connectorItem struct{ ID, Title, Markdown, URL string }
type connectorPage struct {
	Items    []connectorItem
	Next     string
	Warnings []string
}
type connectorRetry struct{ delay time.Duration }

func (e connectorRetry) Error() string {
	return "원격 API가 호출 제한을 반환했습니다. Retry-After 이후 재시도합니다"
}
func (e connectorRetry) RetryAfter() time.Duration { return e.delay }

func connectorURL(base, ref string) (*url.URL, error) {
	root, e := url.Parse(base)
	if e != nil || root.User != nil || !oneOf(root.Scheme, "https", "http") || root.Hostname() == "" || root.Fragment != "" {
		return nil, errors.New("연결 API 주소를 확인하세요")
	}
	if ref == "" {
		return root, nil
	}
	next, e := url.Parse(ref)
	if e != nil || next.User != nil || next.Fragment != "" {
		return nil, errors.New("페이지네이션 URL을 확인하세요")
	}
	if !next.IsAbs() && !strings.HasPrefix(ref, "/") {
		root.Path = strings.TrimSuffix(root.Path, "/") + "/"
	}
	next = root.ResolveReference(next)
	if !strings.EqualFold(next.Host, root.Host) || next.Scheme != root.Scheme {
		return nil, errors.New("원격 API가 다른 출처의 페이지네이션 주소를 반환했습니다")
	}
	return next, nil
}
func (s *Server) connectorHTTP(ctx context.Context, c connectorConfig) (*http.Client, error) {
	var enabled bool
	var raw []byte
	if s.DB.QueryRow(ctx, "SELECT enabled,allowed_hosts FROM connector_settings WHERE id=1").Scan(&enabled, &raw) != nil || !enabled {
		return nil, jobPermanent("관리자가 외부 커넥터 사용을 허용하지 않았습니다")
	}
	var allowed []string
	if json.Unmarshal(raw, &allowed) != nil {
		return nil, errors.New("커넥터 호스트 정책을 읽을 수 없습니다")
	}
	base, e := connectorURL(c.BaseURL, "")
	if e != nil {
		return nil, e
	}
	host := strings.ToLower(base.Hostname())
	found := false
	for _, h := range allowed {
		if strings.EqualFold(h, host) {
			found = true
		}
	}
	if !found {
		return nil, jobPermanent("관리자 허용 목록에 연결 호스트가 없습니다")
	}
	if base.Scheme == "http" && !boolean(c.Config, "allow_http") {
		return nil, jobPermanent("암호화되지 않은 HTTP 연결은 명시적으로 허용해야 합니다")
	}
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: boolean(c.Config, "insecure_tls")}
	if pem := str(c.Config, "ca_pem"); pem != "" {
		pool, _ := x509.SystemCertPool()
		if pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(pem)) {
			return nil, errors.New("CA 인증서 형식을 확인하세요")
		}
		tlsConfig.RootCAs = pool
	}
	transport := &http.Transport{Proxy: nil, TLSClientConfig: tlsConfig, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second, DisableKeepAlives: true, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		dialHost, _, e := net.SplitHostPort(address)
		if e != nil || !strings.EqualFold(dialHost, host) {
			return nil, errors.New("허용되지 않은 커넥터 연결 대상입니다")
		}
		return webhookDial(ctx, network, address)
	}}
	return &http.Client{Transport: transport, Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}
func (s *Server) connectorRead(ctx context.Context, c connectorConfig, ref string) ([]byte, http.Header, error) {
	endpoint, e := connectorURL(c.BaseURL, ref)
	if e != nil {
		return nil, nil, e
	}
	client, e := s.connectorHTTP(ctx, c)
	if e != nil {
		return nil, nil, e
	}
	request, e := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if e != nil {
		return nil, nil, errors.New("커넥터 요청 주소가 올바르지 않습니다")
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "madi-connector/"+s.Version)
	switch str(c.Config, "auth_mode") {
	case "basic":
		request.SetBasicAuth(str(c.Credentials, "username"), str(c.Credentials, "password"))
	case "private_token":
		request.Header.Set("PRIVATE-TOKEN", str(c.Credentials, "token"))
	default:
		if token := str(c.Credentials, "token"); token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
	}
	if c.Kind == "github" {
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	response, e := client.Do(request)
	if e != nil {
		return nil, nil, errors.New("원격 API 연결 실패: 호스트·네트워크·TLS 설정을 확인하세요")
	}
	defer response.Body.Close()
	if response.StatusCode == 429 {
		delay := time.Minute
		if n, e := strconv.Atoi(response.Header.Get("Retry-After")); e == nil && n >= 0 {
			delay = time.Duration(n) * time.Second
		} else if date, e := http.ParseTime(response.Header.Get("Retry-After")); e == nil {
			delay = time.Until(date)
		}
		if delay < time.Second {
			delay = time.Second
		}
		if delay > time.Hour {
			delay = time.Hour
		}
		return nil, nil, connectorRetry{delay}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("원격 API HTTP %d: 인증·읽기 권한·API 경로를 확인하세요", response.StatusCode)
	}
	data, e := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if e != nil || len(data) > 8<<20 {
		return nil, nil, errors.New("원격 API 응답은 8MB 이하여야 합니다")
	}
	return data, response.Header, nil
}
func connectorJSON(raw []byte) (map[string]any, error) {
	var v map[string]any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&v) != nil {
		return nil, errors.New("원격 API JSON 객체를 읽을 수 없습니다")
	}
	return v, nil
}
func connectorValue(v any, path string) any {
	if path == "" {
		return v
	}
	for _, key := range strings.Split(path, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[key]
	}
	return v
}
func connectorString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return ""
}
func connectorArray(v any) []any { x, _ := v.([]any); return x }
func connectorTrim(text string, max int) string {
	if len(text) <= max {
		return text
	}
	v := []rune(text)
	for len(string(v)) > max {
		v = v[:len(v)-1]
	}
	return string(v)
}
func connectorBody(v any) string {
	if text, ok := v.(string); ok {
		return text
	}
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	var b strings.Builder
	if text := str(m, "text"); text != "" {
		b.WriteString(text)
	}
	for _, child := range connectorArray(m["content"]) {
		b.WriteString(connectorBody(child))
	}
	switch str(m, "type") {
	case "paragraph", "heading", "listItem":
		b.WriteByte('\n')
	}
	return b.String()
}
